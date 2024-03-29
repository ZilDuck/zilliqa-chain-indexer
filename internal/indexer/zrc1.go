package indexer

import (
	"encoding/json"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"strconv"
)

type Zrc1Indexer interface {
	IndexTxs(txs []entity.Transaction) error
	IndexTx(tx entity.Transaction, c entity.Contract) error
	IndexContract(c entity.Contract) error
}

type zrc1Indexer struct {
	elastic           elastic_search.Index
	contractRepo      repository.ContractRepository
	contractStateRepo repository.ContractStateRepository
	nftRepo           repository.NftRepository
	txRepo            repository.TransactionRepository
	factory           factory.Zrc1Factory
}

func NewZrc1Indexer(
	elastic elastic_search.Index,
	contractRepo repository.ContractRepository,
	contractStateRepo repository.ContractStateRepository,
	nftRepo repository.NftRepository,
	txRepo repository.TransactionRepository,
	factory factory.Zrc1Factory,
) Zrc1Indexer {
	return zrc1Indexer{elastic, contractRepo, contractStateRepo, nftRepo, txRepo, factory}
}

func (i zrc1Indexer) IndexTxs(txs []entity.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		eventLogs := tx.GetZrc1EventLogs()
		if len(eventLogs) == 0 {
			continue
		}

		c, err := i.contractRepo.GetContractByAddress(eventLogs[0].Address)
		if err != nil {
			continue
		}

		if !c.MatchesStandard(entity.ZRC1) {
			continue
		}

		if err := i.IndexTx(tx, *c); err != nil {
			return err
		}

		i.elastic.BatchPersist()
	}

	return nil
}

func (i zrc1Indexer) IndexTx(tx entity.Transaction, c entity.Contract) error {
	if !c.MatchesStandard(entity.ZRC1) {
		return nil
	}

	zap.L().With(zap.String("txId", tx.ID), zap.String("contract", c.Address)).Debug("Zrc1Indexer: IndexTx")

	if err := i.mint(tx, c); err != nil {
		return err
	}
	if err := i.duckRegeneration(tx, c); err != nil {
		return err
	}
	if err := i.updateTokenUris(tx, c); err != nil {
		return err
	}
	if err := i.transferFrom(tx, c); err != nil {
		return err
	}
	if err := i.burn(tx, c); err != nil {
		return err
	}

	return nil
}

func (i zrc1Indexer) IndexContract(c entity.Contract) error {
	if !c.MatchesStandard(entity.ZRC1) {
		return nil
	}

	size := 100
	page := 1
	for {
		txs, total, err := i.txRepo.GetContractExecutionsByContract(c, size, page)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("contract", c.Address)).Error("Failed to get contract executions")
			return err
		}
		if len(txs) == 0 {
			break
		}
		if page == 1 {
			zap.S().Debugf("Found %d contract executions", total)
		}

		for _, tx := range txs {
			if err := i.IndexTx(tx, c); err != nil {
				return err
			}
		}
		i.elastic.BatchPersist()
		page++
	}

	return nil
}

func (i zrc1Indexer) mint(tx entity.Transaction, c entity.Contract) error {
	nfts, err := i.factory.CreateFromMintTx(tx, c)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create zrc1 from minting tx")
		return err
	}

	for idx := range nfts {
		zap.L().With(zap.String("txID", tx.ID), zap.String("contract", c.Address), zap.Uint64("tokenId", nfts[idx].TokenId)).Info("Mint ZRC1")
		if exists := i.nftRepo.Exists(nfts[idx].Contract, nfts[idx].TokenId); !exists {
			i.elastic.AddIndexRequest(elastic_search.NftIndex.Get(), nfts[idx], elastic_search.Zrc1Mint)
		}
		i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMintAction(nfts[idx]), elastic_search.NftAction)
	}

	return err
}

func (i zrc1Indexer) duckRegeneration(tx entity.Transaction, c entity.Contract) error {
	for _, transition := range tx.GetTransition("regenerateDuck") {
		if !transition.Msg.Params.HasParam("token_id") {
			continue
		}
		tokenId, err := factory.GetTokenId(transition.Msg.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID), zap.String("contract", c.Address)).Debug("Failed to get token id for zrc1:regenerateDuck")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(zap.Uint64("tokenId", tokenId)).Error("Failed to get the nft from the index on duck regeneration")
			return err
		}

		newDuckUri, err := transition.Msg.Params.GetParam("new_duck_uri")
		if err != nil {
			zap.L().Error("Failed to get the new duck metadata on duck regeneration")
			return err
		}
		assetUri := helper.GetIpfs(newDuckUri.Value.String(), nil)
		nft.AssetUri = *assetUri

		newDuckMetaData, err := transition.Msg.Params.GetParam("new_duck_metadata")
		if err != nil {
			zap.L().Error("Failed to get the new duck metadata on duck regeneration")
			return err
		}
		nft.TokenUri = newDuckMetaData.Value.String()
		nft.Metadata = factory.GetMetadata(*nft)

		zap.L().With(zap.String("txID", tx.ID), zap.String("contract", c.Address), zap.Uint64("tokenId", nft.TokenId)).Info("Regenerate NFD")
		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc1DuckRegeneration)
	}

	return nil
}

func (i zrc1Indexer) updateTokenUris(tx entity.Transaction, c entity.Contract) error {

	for _, event := range tx.GetEventLogs(entity.ZRC1MorphTokenURIsUpdated) {
		tokenIdsParam, err := event.Params.GetParam("token_ids")
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get token_ids from ZRC1MorphTokenURIsUpdated")
			continue
		}
		var tokenIds []string

		err = json.Unmarshal([]byte(tokenIdsParam.Value.String()), &tokenIds)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to unmarshal token_ids from ZRC1MorphTokenURIsUpdated")
			continue
		}

		state, err := i.contractStateRepo.GetStateByAddress(c.Address)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contract state from ZRC1MorphTokenURIsUpdated")
			continue
		}
		baseUri, exists := state.GetElement("base_uri")
		if !exists {
			zap.L().With(zap.Error(err)).Error("Failed to get baseUri from ZRC1MorphTokenURIsUpdated")
			continue
		}

		for _, tokenIdString := range tokenIds {
			tokenId, err := strconv.ParseUint(tokenIdString, 10, 64)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("tokenId", tokenIdString)).Error("Failed to parse token_id from ZRC1MorphTokenURIsUpdated")
				continue
			}
			nft, err := i.nftRepo.GetNft(c.Address, tokenId)
			if err != nil {
				zap.L().With(zap.Error(err), zap.Uint64("tokenId", tokenId)).Error("Failed to find nft from ZRC1MorphTokenURIsUpdated")
				continue
			}

			oldTokenUri := nft.TokenUri
			nft.TokenUri = fmt.Sprintf("%s%d", baseUri, tokenId)
			nft.Metadata.Uri = factory.GetMetadataUri(*nft)

			zap.L().With(
				zap.String("contract", c.Address),
				zap.Uint64("tokenId", nft.TokenId),
				zap.String("from", oldTokenUri),
				zap.String("to", nft.TokenUri),
			).Info("Update token URI")
			i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc1UpdateTokenUri)
		}
	}

	return nil
}

func (i zrc1Indexer) transferFrom(tx entity.Transaction, c entity.Contract) error {
	var eventName entity.Event
	if tx.HasEventLog(entity.ZRC1TransferEvent) {
		eventName = entity.ZRC1TransferEvent
	} else if tx.HasEventLog(entity.ZRC1TransferFromEvent) {
		eventName = entity.ZRC1TransferFromEvent
	} else {
		return nil
	}

	for _, event := range tx.GetEventLogs(eventName) {
		tokenId, err := factory.GetTokenId(event.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID), zap.String("contract", c.Address)).Debug("Failed to get token id for zrc1:transfer")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.String("txId", tx.ID),
				zap.String("contract", c.Address),
				zap.Uint64("tokenId", tokenId),
				zap.String("action", "transfer"),
			).Error("Failed to find zrc1 nft in index")
			continue
		}

		prevOwner, err := event.Params.GetParam("from")
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contract", c.Address)).Error("Failed to get zrc1:from for transfer")
			return err
		}

		newOwner, err := event.Params.GetParam("recipient")
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contract", c.Address)).Error("Failed to get zrc1:recipient for transfer")
			return err
		}

		txType := tx.GetMarketplaceTxType()
		if txType != "" {
			if txType == "listing" {
				nft.IsDelegated = true
				nft.DelegatedOwner = newOwner.Value.String()
			} else {
				nft.IsDelegated = false
				nft.DelegatedOwner = ""
			}
			i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftDelegate)
		} else {

			oldOwner := nft.Owner
			nft.Owner = newOwner.Value.String()

			zap.L().With(
				zap.String("txID", tx.ID),
				zap.String("contract", nft.Contract),
				zap.Uint64("tokenId", nft.TokenId),
				zap.String("from", oldOwner),
				zap.String("to", nft.Owner),
			).Info("Transfer ZRC1")

			i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc1Transfer)
			i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateTransferAction(*nft, tx.BlockNum, tx.ID, nft.Owner, prevOwner.Value.String()), elastic_search.Zrc1Transfer)
		}
	}

	return nil
}

func (i zrc1Indexer) burn(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasEventLog(entity.ZRC1BurnEvent) {
		return nil
	}
	zap.L().With(zap.String("txId", tx.ID), zap.String("contract", c.Address)).Debug("Zrc1Indexer: burn")

	for _, event := range tx.GetEventLogs(entity.ZRC1BurnEvent) {
		tokenId, err := factory.GetTokenId(event.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID), zap.String("contract", c.Address)).Warn("Failed to get token id for zrc1:burn")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.String("contract", c.Address),
				zap.Uint64("tokenId", tokenId),
				zap.String("action", "burn"),
			).Error("Failed to find zrc1 nft in index")
			continue
		}
		nft.BurnedAt = tx.BlockNum

		zap.L().With(zap.String("txID", tx.ID), zap.String("contract", c.Address), zap.Uint64("tokenId", nft.TokenId)).Info("Burn ZRC1")

		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.Zrc1Burn)
		i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateBurnAction(*nft, tx), elastic_search.Zrc1Burn)
	}

	return nil
}
