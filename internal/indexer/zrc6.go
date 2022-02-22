package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type Zrc6Indexer interface {
	IndexTxs(tx []entity.Transaction) error
	IndexTx(tx entity.Transaction, c entity.Contract) error
	IndexContract(c entity.Contract) error

	TriggerMetadataRefresh(nft entity.Nft)
	TriggerAssetRefresh(nft entity.Nft)

	RefreshMetadata(contractAddr string, tokenId uint64) error
	RefreshAsset(contractAddr string, tokenId uint64) error
}

type zrc6Indexer struct {
	elastic         elastic_search.Index
	contractRepo    repository.ContractRepository
	nftRepo         repository.NftRepository
	txRepo          repository.TransactionRepository
	factory         factory.Zrc6Factory
	messageService  messenger.MessageService
	metadataService metadata.Service
}

func NewZrc6Indexer(
	elastic elastic_search.Index,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
	txRepo repository.TransactionRepository,
	factory factory.Zrc6Factory,
	messageService messenger.MessageService,
	metadataService metadata.Service,
) Zrc6Indexer {
	return zrc6Indexer{elastic, contractRepo, nftRepo, txRepo,factory, messageService, metadataService}
}

func (i zrc6Indexer) IndexTxs(txs []entity.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		transitions := tx.GetZrc6Transitions()
		if len(transitions) == 0 {
			continue
		}

		c, err := i.contractRepo.GetContractByAddress(transitions[0].Addr)
		if err != nil {
			continue
		}

		if err := i.IndexTx(tx, *c); err != nil {
			return err
		}

		i.elastic.BatchPersist()
	}

	return nil
}

func (i zrc6Indexer) IndexTx(tx entity.Transaction, c entity.Contract) error {
	zap.S().With(zap.String("contractAddr", c.Address)).Infof("Index ZRC6 From TX %s", tx.ID)
	if !c.ZRC6 {
		return nil
	}

	if err := i.mint(tx, c); err != nil {
		return err
	}
	if err := i.batchMint(tx, c); err != nil {
		return err
	}
	if err := i.setBaseUri(tx, c); err != nil {
		return err
	}
	if err := i.transferFrom(tx, c); err != nil {
		return err
	}
	if err := i.burn(tx, c); err != nil {
		return err
	}
	if err := i.batchBurn(tx, c); err != nil {
		return err
	}

	return nil
}

func (i zrc6Indexer) IndexContract(c entity.Contract) error {
	if !c.ZRC6 {
		return nil
	}

	size := 100
	page := 1
	for {
		txs, _, err := i.txRepo.GetContractExecutionsByContract(c, size, page)
		if err != nil {
			return err
		}
		if len(txs) == 0 {
			zap.L().Info("No more txs")
			break
		}

		for _, tx := range txs {
			if err := i.IndexTx(tx, c); err != nil {
				return err
			}
		}
		page++
		i.elastic.BatchPersist()
	}

	return nil
}

func (i zrc6Indexer) mint(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasEventLog(entity.ZRC6MintEvent) {
		return nil
	}

	nfts, err := i.factory.CreateFromMintTx(tx, c)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create zrc6 from minting tx")
		return err
	}

	for idx := range nfts {
		i.elastic.AddIndexRequest(elastic_search.NftIndex.Get(), nfts[idx], elastic_search.Zrc6Mint)

		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nfts[idx].TokenId),
			zap.String("owner", nfts[idx].Owner),
		).Info("Mint ZRC6")

		i.TriggerMetadataRefresh(nfts[idx])
	}

	return nil
}

func (i zrc6Indexer) batchMint(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasEventLog(entity.ZRC6BatchMintEvent) {
		return nil
	}

	nfts, err := i.factory.CreateFromBatchMint(tx, c)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create zrc6 from batch minting tx")
		return err
	}

	for idx := range nfts {
		i.elastic.AddIndexRequest(elastic_search.NftIndex.Get(), nfts[idx], elastic_search.Zrc6Mint)

		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nfts[idx].TokenId),
			zap.String("owner", nfts[idx].Owner),
		).Info("BatchMint ZRC6")

		i.TriggerMetadataRefresh(nfts[idx])
	}

	return nil
}

func (i zrc6Indexer) setBaseUri(tx entity.Transaction, c entity.Contract) error {
	for _, event := range tx.GetEventLogs(entity.ZRC6SetBaseURIEvent) {
		baseUri, err := event.Params.GetParam("base_uri")
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to get zrc6:base_uri from ZRC6SetBaseURIEvent")
			return err
		}
		c.BaseUri = baseUri.Value.Primitive.(string)

		i.elastic.AddUpdateRequest(elastic_search.ContractIndex.Get(), c, elastic_search.ContractSetBaseUri)
		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("blockNum", tx.BlockNum), zap.String("baseUri", c.BaseUri)).Info("Update Contract base uri")

		size := 100
		page := 1
		for {
			nfts, _, err := i.nftRepo.GetNfts(c.Address, size, page)
			if err != nil {
				return err
			}
			if len(nfts) == 0 {
				break
			}

			for _, nft := range nfts {
				nft.BaseUri = c.BaseUri
				i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.Zrc6SetBaseUri)
			}

			i.elastic.BatchPersist()
			page++
		}
	}

	return nil
}

func (i zrc6Indexer) transferFrom(tx entity.Transaction, c entity.Contract) error {
	for _, event := range tx.GetEventLogs(entity.ZRC6TransferFromEvent) {
		tokenId, err := factory.GetTokenId(event.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("contractAddr", c.Address)).
				Warn("Failed to get token id for zrc6:transfer")
			continue
		}

		to, err := event.Params.GetParam("to")
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get zrc6:new_owner")
			return err
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.String("txId", tx.ID),
				zap.String("contractAddr", c.Address),
				zap.Uint64("tokenId", tokenId),
			).Error("Failed to find nft in index")
			continue
		}

		nft.Owner = to.Value.Primitive.(string)

		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc6Transfer)
		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nft.TokenId),
			zap.String("to", nft.Owner),
		).Info("Transfer ZRC6")
	}

	return nil
}

func (i zrc6Indexer) burn(tx entity.Transaction, c entity.Contract) error {
	for _, event := range tx.GetEventLogs(entity.ZRC6BurnEvent) {
		tokenId, err := factory.GetTokenId(event.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("contractAddr", c.Address)).
				Warn("Failed to get token id for zrc6:transfer")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.String("txId", tx.ID),
				zap.String("contractAddr", c.Address),
				zap.Uint64("tokenId", tokenId),
			).Error("Failed to find nft in index")
			continue
		}

		nft.BurnedAt = tx.BlockNum

		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc6Burn)
		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nft.TokenId),
		).Info("Burn ZRC6")
	}

	return nil
}

func (i zrc6Indexer) batchBurn(tx entity.Transaction, c entity.Contract) error {
	for range tx.GetTransition(entity.ZRC6BatchBurnCallback) {

	}

	return nil
}

func (i zrc6Indexer) TriggerMetadataRefresh(nft entity.Nft) {
	if nft.Metadata == nil {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.MetadataRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue metadata refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger MetaData Refresh")
}

func (i zrc6Indexer) TriggerAssetRefresh(nft entity.Nft) {
	if nft.Metadata == nil {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.AssetRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue asset refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger Asset Refresh")
}

func (i zrc6Indexer) RefreshMetadata(contractAddr string, tokenId uint64) error {
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId)).Info("ZRC6 Refresh Metadata")
	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		return err
	}

	data, err := i.metadataService.FetchZrc6Metadata(*nft)
	if err != nil {
		zap.L().With(
			zap.Error(err),
			zap.String("contractAddr", nft.Contract),
			zap.Uint64("tokenId", nft.TokenId),
			zap.String("baseUrl", nft.BaseUri),
			zap.String("tokenUri", nft.TokenUri),
		).Warn("Failed to get zrc6 metadata")
		if nft.Metadata != nil {
			nft.Metadata.Error = err.Error()
			i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.Zrc6Metadata)
			i.elastic.BatchPersist()
		}
		return err
	}

	nft.Metadata.Data = data
	nft.Metadata.Error = ""

	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.Zrc6Metadata)
	i.elastic.BatchPersist()

	return nil
}

func (i zrc6Indexer) RefreshAsset(contractAddr string, tokenId uint64) error {
	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		return err
	}
	if nft.Zrc6 == true {
		err := i.metadataService.FetchZrc6Image(*nft)
		if err != nil {
			if errors.Is(err, metadata.ErrorAssetAlreadyExists) {
				//return nil
			} else {
				zap.L().With(zap.Error(err)).Error("Failed to fetch zrc6 asset")
				return err
			}
		}

		nft.MediaUri = fmt.Sprintf("%s/%d", contractAddr, tokenId)
		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.Zrc6Asset)
		i.elastic.BatchPersist()
	}

	return nil
}
