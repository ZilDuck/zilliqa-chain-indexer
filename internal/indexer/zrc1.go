package indexer

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_cache"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type Zrc1Indexer interface {
	IndexTxs(txs []entity.Transaction) error
	IndexTx(tx entity.Transaction, c entity.Contract) error
	IndexContract(c entity.Contract) error
}

type zrc1Indexer struct {
	elastic      elastic_cache.Index
	contractRepo repository.ContractRepository
	nftRepo      repository.NftRepository
	txRepo       repository.TransactionRepository
}

func NewZrc1Indexer(
	elastic elastic_cache.Index,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
	txRepo repository.TransactionRepository,
) Zrc1Indexer {
	return zrc1Indexer{elastic, contractRepo, nftRepo, txRepo}
}

func (i zrc1Indexer) IndexTxs(txs []entity.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		transitions := tx.GetZrc1Transitions()
		if len(transitions) == 0 {
			continue
		}

		if c, err := i.contractRepo.GetContractByAddress(transitions[0].Addr); err != nil {
			if err := i.IndexTx(tx, c); err != nil {
				return err
			}
		}
		i.elastic.BatchPersist()
	}

	return nil
}

func (i zrc1Indexer) IndexTx(tx entity.Transaction, c entity.Contract) error {
	if !c.ZRC1 {
		return nil
	}

	if err := i.mint(tx, c); err != nil {
		return err
	}
	if err := i.duckRegeneration(tx, c); err != nil {
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
	if !c.ZRC1 {
		return nil
	}

	size := 100
	page := 1
	for {
		txs, total, err := i.txRepo.GetContractExecutionsByContract(c, size, page)
		if err != nil {
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
	nfts, err := factory.CreateZrc1FromMintTx(tx, c)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create zrc1 from minting tx")
		return err
	}

	for idx := range nfts {
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nfts[idx], elastic_cache.Zrc1Mint)

		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nfts[idx].TokenId),
			zap.String("owner", nfts[idx].Owner),
		).Info("Mint ZRC1")
	}

	return err
}

func (i zrc1Indexer) duckRegeneration(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasTransition(entity.ZRC1RegenerateDuck) {
		return nil
	}

	for _, transition := range tx.GetTransition(entity.ZRC1RegenerateDuck) {
		if !transition.Msg.Params.HasParam("token_id", "Uint256") {
			continue
		}
		tokenId, _ := factory.GetTokenId(transition.Msg.Params)

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(zap.Uint64("tokenId", tokenId)).Error("Failed to get zrc1 from the index on duck regeneration")
			return err
		}

		newDuckMetaData, err := transition.Msg.Params.GetParam("new_duck_metadata")
		if err != nil {
			zap.L().Error("Failed to get zrc1:new_duck_metadata on duck regeneration")
			return err
		}

		nft.TokenUri = newDuckMetaData.Value.Primitive.(string)
		zap.L().With(
			zap.Uint64("blockNum", tx.BlockNum),
			zap.String("symbol", nft.Symbol),
			zap.Uint64("tokenId", nft.TokenId),
		).Info("Regenerate ZRC1 Duck")

		i.elastic.AddUpdateRequest(elastic_cache.NftIndex.Get(), nft, elastic_cache.Zrc1DuckRegeneration)
	}

	return nil
}

func (i zrc1Indexer) transferFrom(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasTransition(entity.ZRC1RecipientAcceptTransfer) {
		return nil
	}

	for _, transition := range tx.GetTransition(entity.ZRC1RecipientAcceptTransfer) {
		tokenId, err := factory.GetTokenId(transition.Msg.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID), zap.String("contractAddr", c.Address)).Warn("Failed to get token id for zrc1:transfer")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("contract", c.Address), zap.Uint64("tokenId", tokenId)).Error("Failed to find nft in index")
		}

		previousOwner := nft.Owner

		newOwner, err := transition.Msg.Params.GetParam("recipient")
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contractAddr", c.Address)).Error("Failed to get zrc1:recipient for transfer")
			return err
		}

		nft.Owner = newOwner.Value.Primitive.(string)

		zap.L().With(
			zap.Uint64("blockNum", tx.BlockNum),
			zap.String("symbol", nft.Symbol),
			zap.Uint64("tokenId", nft.TokenId),
			zap.String("from", previousOwner),
			zap.String("to", nft.Owner),
		).Info("Transfer ZRC1")

		i.elastic.AddUpdateRequest(elastic_cache.NftIndex.Get(), nft, elastic_cache.Zrc1Transfer)
	}

	return nil
}

func (i zrc1Indexer) burn(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasTransition(entity.ZRC1BurnCallBack) {
		return nil
	}

	for _, transition := range tx.GetTransition(entity.ZRC1BurnCallBack) {
		tokenId, err := factory.GetTokenId(transition.Msg.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID), zap.String("contractAddr", c.Address)).Warn("Failed to get token id for zrc1:transfer")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("contract", c.Address), zap.Uint64("tokenId", tokenId)).Error("Failed to find nft in index")
		}

		nft.BurnedAt = tx.BlockNum

		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nft.TokenId),
		).Info("Burn ZRC1")

		i.elastic.AddUpdateRequest(elastic_cache.NftIndex.Get(), nft, elastic_cache.Zrc1Burn)
	}

	return nil
}
