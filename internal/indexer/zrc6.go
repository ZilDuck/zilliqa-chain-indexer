package indexer

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_cache"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type Zrc6Indexer interface {
	IndexTxs(tx []entity.Transaction) error
	IndexTx(tx entity.Transaction, c entity.Contract) error
	IndexContract(c entity.Contract) error
}

type zrc6Indexer struct {
	elastic       elastic_cache.Index
	contractRepo  repository.ContractRepository
	nftRepo       repository.NftRepository
	txRepo        repository.TransactionRepository
	latestTokenId map[string]uint64
}

func NewZrc6Indexer(
	elastic elastic_cache.Index,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
	txRepo repository.TransactionRepository,
) Zrc6Indexer {
	return zrc6Indexer{elastic, contractRepo, nftRepo, txRepo, map[string]uint64{}}
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

		if c, err := i.contractRepo.GetContractByAddress(transitions[0].Addr); err != nil {
			if err := i.IndexTx(tx, c); err != nil {
				return err
			}
		}
		i.elastic.BatchPersist()
	}

	return nil
}

func (i zrc6Indexer) IndexTx(tx entity.Transaction, c entity.Contract) error {
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
	if !tx.HasTransition(entity.ZRC6MintCallback) {
		return nil
	}

	nfts, err := factory.CreateZrc6FromMintTx(tx, c)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create zrc6 from minting tx")
		return err
	}

	for idx := range nfts {
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nfts[idx], elastic_cache.Zrc6Mint)
		i.latestTokenId[c.Address] = nfts[idx].TokenId

		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nfts[idx].TokenId),
			zap.String("owner", nfts[idx].Owner),
		).Info("Mint ZRC6")
	}

	return nil
}

func (i zrc6Indexer) batchMint(tx entity.Transaction, c entity.Contract) error {
	if !tx.HasTransition(entity.ZRC6BatchMintCallback) {
		return nil
	}

	bestTokenId, exists := i.latestTokenId[c.Address]
	if !exists {
		zap.L().With(zap.String("txID", tx.ID), zap.String("contractAddr", c.Address)).Warn("Getting best token id from index")
		var err error
		bestTokenId, err = i.nftRepo.GetBestTokenId(c.Address, tx.BlockNum)
		if err != nil {
			return err
		}
	}

	nfts, err := factory.CreateZrc6FromBatchMint(tx, c, bestTokenId+1)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create zrc6 from batch minting tx")
		return err
	}

	for idx := range nfts {
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nfts[idx], elastic_cache.Zrc6Mint)
		i.latestTokenId[c.Address] = nfts[idx].TokenId

		zap.L().With(
			zap.String("contractAddr", c.Address),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.Uint64("tokenId", nfts[idx].TokenId),
			zap.String("owner", nfts[idx].Owner),
		).Info("BatchMint ZRC6")
	}

	return nil
}

func (i zrc6Indexer) setBaseUri(tx entity.Transaction, c entity.Contract) error {
	for _, t := range tx.GetTransition(entity.ZRC6SetBaseURICallback) {
		baseUri, err := t.Msg.Params.GetParam("base_uri")
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to get zrc6:base_uri from BaseUriCallback")
			return err
		}

		c.BaseUri = baseUri.Value.Primitive.(string)

		i.elastic.AddUpdateRequest(elastic_cache.ContractIndex.Get(), c, elastic_cache.ContractSetBaseUri)
		zap.L().With(zap.String("contractAddr", c.Address), zap.Uint64("blockNum", tx.BlockNum), zap.String("baseUri", c.BaseUri)).Info("Update Contract base uri")

		size := 100
		page := 1
		for {
			nfts, _, err := i.nftRepo.GetNfts(t.Addr, size, page)
			if err != nil {
				return err
			}
			if len(nfts) == 0 {
				break
			}

			for _, nft := range nfts {
				nft.TokenUri = c.BaseUri
				i.elastic.AddUpdateRequest(elastic_cache.NftIndex.Get(), nft, elastic_cache.Zrc6SetBaseUri)
			}

			i.elastic.BatchPersist()
			page++
		}
	}

	return nil
}

func (i zrc6Indexer) transferFrom(tx entity.Transaction, c entity.Contract) error {
	for _, transition := range tx.GetTransition(entity.ZRC6RecipientAcceptTransferFrom) {
		if transition.Addr != c.Address {
			continue
		}

		tokenId, err := factory.GetTokenId(transition.Msg.Params)
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.String("contractAddr", c.Address),
			).Warn("Failed to get token id for zrc6:transfer")
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

		to, err := transition.Msg.Params.GetParam("to")
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get zrc6:new_owner")
			return err
		}

		nft.Owner = to.Value.Primitive.(string)

		i.elastic.AddUpdateRequest(elastic_cache.NftIndex.Get(), nft, elastic_cache.Zrc6Transfer)
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
	for _, transition := range tx.GetTransition(entity.ZRC6BurnCallback) {
		if transition.Addr != c.Address {
			continue
		}

		tokenId, err := factory.GetTokenId(transition.Msg.Params)
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.String("contractAddr", c.Address),
			).Warn("Failed to get token id for zrc6:transfer")
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

		i.elastic.AddUpdateRequest(elastic_cache.NftIndex.Get(), nft, elastic_cache.Zrc6Burn)
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
