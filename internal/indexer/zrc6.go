package indexer

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"strconv"
)

type Zrc6Indexer interface {
	IndexTxs(tx []entity.Transaction) error
	IndexTx(tx entity.Transaction, c entity.Contract) error
	IndexContract(c entity.Contract) error
}

type zrc6Indexer struct {
	elastic         elastic_search.Index
	contractRepo    repository.ContractRepository
	nftRepo         repository.NftRepository
	txRepo          repository.TransactionRepository
	factory         factory.Zrc6Factory
}

func NewZrc6Indexer(
	elastic elastic_search.Index,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
	txRepo repository.TransactionRepository,
	factory factory.Zrc6Factory,
) Zrc6Indexer {
	return zrc6Indexer{elastic, contractRepo, nftRepo, txRepo, factory}
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
	if !c.MatchesStandard(entity.ZRC6) {
		return nil
	}
	zap.L().With(zap.String("contractAddr", c.Address), zap.String("txID", tx.ID)).Debug("Zrc6Indexer: Index ZRC6")

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
	if err := i.setTokenUri(tx, c); err != nil {
		return err
	}
	if err := i.batchSetTokenUri(tx, c); err != nil {
		return err
	}

	return nil
}

func (i zrc6Indexer) IndexContract(c entity.Contract) error {
	if !c.MatchesStandard(entity.ZRC6) {
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
	i.elastic.Persist()

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
		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("tokenId", nfts[idx].TokenId)).Info("Mint ZRC6")
		if exists := i.nftRepo.Exists(nfts[idx].Contract, nfts[idx].TokenId); !exists {
			i.elastic.AddIndexRequest(elastic_search.NftIndex.Get(), nfts[idx], elastic_search.Zrc6Mint)
		}
		i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMintAction(nfts[idx]), elastic_search.NftAction)
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
		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("tokenId", nfts[idx].TokenId)).Info("BatchMint ZRC6")
		if exists := i.nftRepo.Exists(nfts[idx].Contract, nfts[idx].TokenId); !exists {
			i.elastic.AddIndexRequest(elastic_search.NftIndex.Get(), nfts[idx], elastic_search.Zrc6Mint)
		}
		i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMintAction(nfts[idx]), elastic_search.NftAction)
		i.elastic.BatchPersist()
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
				nft.Metadata.Uri = factory.GetMetadataUri(nft)
				i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.Zrc6SetBaseUri)
			}

			i.elastic.BatchPersist()
			page++
		}
	}

	return nil
}

func (i zrc6Indexer) setTokenUri(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasTransition(string(entity.ZRC6SetTokenURICallback)) {
		return nil
	}
	if tx.Data.Tag != "UpdateTokenUri" {
		return nil
	}

	tokenId, err := tx.Data.Params.GetParam("token_id")
	if err != nil {
		zap.L().With(zap.String("contractAddr", c.Address)).Error("Failed to get token_id_token_uri_pair_list from BatchUpdateTokenUri")
		return nil
	}

	tokenUri, err := tx.Data.Params.GetParam("new_uri")
	if err != nil {
		zap.L().With(zap.String("contractAddr", c.Address)).Error("Failed to get token_id_token_uri_pair_list from BatchUpdateTokenUri")
		return nil
	}

	tokenIdInt, err := strconv.Atoi(tokenId.Value.Primitive.(string))
	if err != nil {
		return nil
	}

	nft, err := i.nftRepo.GetNft(c.Address, uint64(tokenIdInt))
	if err != nil {
		return nil
	}
	nft.TokenUri = tokenUri.Value.Primitive.(string)

	zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("tokenId", nft.TokenId)).Info("Update token URI")
	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc6SetTokenUri)

	return nil
}

func (i zrc6Indexer) batchSetTokenUri(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasTransition(string(entity.ZRC6BatchSetTokenURICallback)) {
		return nil
	}
	if tx.Data.Tag != "BatchUpdateTokenUri" {
		return nil
	}

	type TokenIdTokenUriPairList struct {
		ArgTypes  []string `json:"argtypes"`
		Arguments []string `json:"arguments"`
		Constructor string `json:"constructor"`
	}

	data, err := tx.Data.Params.GetParam("token_id_token_uri_pair_list")
	if err != nil {
		zap.L().With(zap.String("contractAddr", c.Address)).Error("Failed to get token_id_token_uri_pair_list from BatchUpdateTokenUri")
		return nil
	}

	var tokenIdTokenUriPairList []TokenIdTokenUriPairList
	if err := json.Unmarshal([]byte(data.Value.Primitive.(string)), &tokenIdTokenUriPairList); err != nil {
		zap.L().With(zap.String("contractAddr", c.Address)).Error("Failed to unmarshall token_id_token_uri_pair_list from BatchUpdateTokenUri")
	}

	for _, tokenIdTokenUriPair := range tokenIdTokenUriPairList {
		tokenId, err := strconv.Atoi(tokenIdTokenUriPair.Arguments[0])
		if err != nil {
			continue
		}
		nft, err := i.nftRepo.GetNft(c.Address, uint64(tokenId))
		if err != nil {
			continue
		}
		nft.TokenUri = tokenIdTokenUriPair.Arguments[1]
		nft.Metadata.Uri = factory.GetMetadataUri(*nft)
		nft.Metadata.IsIpfs = helper.IsIpfs(nft.Metadata.Uri)
		nft.Metadata.Status = entity.MetadataPending
		nft.Metadata.Error = ""

		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("tokenId", nft.TokenId), zap.String("tokenUri", nft.TokenUri)).Info("Update token URI")
		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc6SetTokenUri)
	}

	return nil
}

func (i zrc6Indexer) transferFrom(tx entity.Transaction, c entity.Contract) error {
	if tx.IsMarketplaceTx() {
		return nil
	}

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
			).Fatal("Failed to find zrc6 nft in index")
			continue
		}

		prevOwner, err := event.Params.GetParam("from")
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contractAddr", c.Address)).Error("Failed to get zrc1:from for transfer")
			return err
		}

		nft.Owner = to.Value.Primitive.(string)

		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("tokenId", nft.TokenId)).Info("Transfer ZRC6")

		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc6Transfer)
		i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateTransferAction(*nft, tx.BlockNum, tx.ID, nft.Owner, prevOwner.Value.String()), elastic_search.Zrc6Transfer)
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
			).Error("Failed to find zrc6 nft in index")
			continue
		}

		nft.BurnedAt = tx.BlockNum

		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("tokenId", nft.TokenId)).Info("Burn ZRC6")

		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc6Burn)
		i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateBurnAction(*nft, tx), elastic_search.Zrc6Burn)
	}

	return nil
}

func (i zrc6Indexer) batchBurn(tx entity.Transaction, c entity.Contract) error {
	return nil
}