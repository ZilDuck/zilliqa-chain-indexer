package indexer

import (
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/entity"
	"github.com/dantudor/zil-indexer/internal/factory"
	"github.com/dantudor/zil-indexer/internal/repository"
	"go.uber.org/zap"
	"log"
	"time"
)

var (
	defaultSize = 100
)

type NftIndexer interface {
	Index(txs []entity.Transaction) error
	BulkIndex(fromBlockNum uint64) error

	IndexTxMints(tx entity.Transaction, c entity.Contract) error
	IndexTxDuckRegenerations(tx entity.Transaction, c entity.Contract) error
	IndexTxTransfers(tx entity.Transaction, c entity.Contract) error

	IndexContract(c entity.Contract) error
	IndexContractMints(c entity.Contract) error
	IndexContractDuckRegenerations(c entity.Contract) error
	IndexContractTransfers(c entity.Contract) error
}

type nftIndexer struct {
	elastic      elastic_cache.Index
	contractRepo repository.ContractRepository
	nftRepo      repository.NftRepository
	txRepo       repository.TransactionRepository
}

func NewNftIndexer(
	elastic elastic_cache.Index,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
	txRepo repository.TransactionRepository,
) NftIndexer {
	return nftIndexer{elastic, contractRepo, nftRepo, txRepo}
}

func (i nftIndexer) Index(txs []entity.Transaction) error {
	contracts := map[string]*entity.Contract{}
	for _, tx := range txs {
		if _, ok := contracts[tx.ContractAddress]; !ok {
			c, _ := i.contractRepo.GetContractByMinterFallbackToAddress(tx.ContractAddress)
			contracts[tx.ContractAddress] = &c
		}
	}

	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		if contracts[tx.ContractAddress].ZRC1 == false {
			continue
		}

		if err := i.IndexTxMints(tx, *contracts[tx.ContractAddress]); err != nil {
			return err
		}

		if err := i.IndexTxDuckRegenerations(tx, *contracts[tx.ContractAddress]); err != nil {
			return err
		}

		if err := i.IndexTxTransfers(tx, *contracts[tx.ContractAddress]); err != nil {
			return err
		}
	}

	return nil
}

func (i nftIndexer) BulkIndex(fromBlockNum uint64) error {
	zap.L().With(zap.Uint64("from", fromBlockNum)).Info("Bulk index nfts")

	size := defaultSize
	page := 1

	for {
		txs, _, err := i.txRepo.GetContractExecutionTxs(fromBlockNum, size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contract txs")
			return err
		}

		if len(txs) == 0 {
			zap.L().Info("No more contract execution txs found")
			break
		}

		if err := i.Index(txs); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to index NFTs")
		}

		i.elastic.BatchPersist()

		page++
	}

	i.elastic.Persist()

	return nil
}

func (i nftIndexer) IndexContract(c entity.Contract) error {
	symbol, _ := c.Data.Params.GetParam("symbol")
	name, _ := c.Data.Params.GetParam("name")

	log.Println("")
	zap.S().With(
		zap.String("name", name.Value.Primitive.(string)),
		zap.String("symbol", symbol.Value.Primitive.(string)),
	).Infof("Indexing NFTs for %s", c.AddressBech32)

	if err := i.IndexContractMints(c); err != nil {
		return err
	}

	if err := i.IndexContractDuckRegenerations(c); err != nil {
		return err
	}

	if err := i.IndexContractTransfers(c); err != nil {
		return err
	}

	return nil
}

func (i nftIndexer) IndexTxMints(tx entity.Transaction, c entity.Contract) error {
	nfts, err := factory.CreateNftsFromMintingTx(tx, c)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create nft from minting tx")
		return err
	}

	for idx := range nfts {
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nfts[idx])
	}
	zap.L().With(zap.Int("count", len(nfts))).Info("Index nft mints")

	return nil
}

func (i nftIndexer) IndexContractMints(c entity.Contract) (err error) {
	zap.L().With(zap.String("contractAddr", c.Address)).Info("Index Contract Mints")

	indexContractMints := func(contractAddr string) error {
		size := defaultSize
		from := 1
		for {
			txs, _, err := i.txRepo.GetContractTxs(contractAddr, size, from)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("contractAddr", c.Address)).Fatal("Failed to get txs for contract")
			}

			for _, tx := range txs {
				if err := i.IndexTxMints(tx, c); err != nil {
					return err
				}
			}

			if len(txs) == 0 {
				break
			}

			from = from + size - 1
			i.elastic.BatchPersist()
		}

		return nil
	}

	if err := indexContractMints(c.Address); err != nil {
		zap.S().Errorf("Failed to collect minted NFTs for contract %s", c.Address)
		return err
	}

	for _, minter := range c.Minters {
		if err := indexContractMints(minter); err != nil {
			zap.S().Errorf("Failed to collect minted NFTs for contract %s by minter %s", c.Address, minter)
			return err
		}
	}

	return
}

func (i nftIndexer) IndexTxDuckRegenerations(tx entity.Transaction, c entity.Contract) error {
	transitions := tx.GetTransition(entity.TransitionRegenerateDuck)
	for _, transition := range transitions {
		if !transition.Msg.Params.HasParam("token_id", "Uint256") {
			continue
		}
		tokenId, _ := factory.GetTokenId(transition.Msg.Params)

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			zap.L().With(zap.Uint64("tokenId", tokenId)).Error("Failed to get the nft from the index on duck regeneration")
			return err
		}

		newDuckMetaData, err := transition.Msg.Params.GetParam("new_duck_metadata")
		if err != nil {
			zap.L().Error("Failed to get the new duck metadata on duck regeneration")
			return err
		}

		nft.TokenUri = newDuckMetaData.Value.Primitive.(string)
		zap.L().With(
			zap.Uint64("blockNum", tx.BlockNum),
			zap.String("symbol", nft.Symbol),
			zap.Uint64("tokenId", nft.TokenId),
		).Info("Regenerate NFD")

		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nft)
	}
	zap.L().With(zap.Int("count", len(transitions))).Info("Index nft duck regenerations")

	return nil
}

func (i nftIndexer) IndexContractDuckRegenerations(c entity.Contract) error {
	if !c.HasTransition(entity.TransitionRegenerateDuck) {
		return nil
	}

	zap.L().Info("Index Duck Regenerations")

	indexDuckRegenerations := func(contractAddr string) error {
		size := defaultSize
		from := 0

		for {
			txs, _, err := i.txRepo.GetContractExecutionsWithTransition(contractAddr, entity.TransitionRegenerateDuck, size, from)
			if err != nil {
				return err
			}

			if len(txs) == 0 {
				break
			}

			for _, tx := range txs {
				if err := i.IndexTxDuckRegenerations(tx, c); err != nil {
					return err
				}
			}

			from = from + size - 1
			i.elastic.BatchPersist()
		}

		return nil
	}

	if err := indexDuckRegenerations(c.Address); err != nil {
		return err
	}
	for _, minter := range c.Minters {
		if err := indexDuckRegenerations(minter); err != nil {
			return err
		}
	}

	return nil
}

func (i nftIndexer) IndexTxTransfers(tx entity.Transaction, c entity.Contract) error {
	err := i.handleTransfersForTx(c, tx)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to handle transfers")
	}
	return err
}

func (i nftIndexer) IndexContractTransfers(c entity.Contract) error {
	zap.L().With(zap.String("contractAddr", c.Address)).Info("Index Contract Transfers")

	size := defaultSize
	from := 0

	for {
		txs, _, err := i.txRepo.GetContractExecutionsWithTransition(c.Address, entity.TransitionRecipientAcceptTransfer, size, from)
		if err != nil {
			return err
		}

		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			err = i.handleTransfersForTx(c, tx)
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to handle transfers")
				return err
			}
		}

		from = from + size - 1
		i.elastic.BatchPersist()
	}

	return nil
}

func (i nftIndexer) handleTransfersForTx(c entity.Contract, tx entity.Transaction) error {
	rats := tx.GetTransition(entity.TransitionRecipientAcceptTransfer)
	for _, rat := range rats {
		tokenId, err := factory.GetTokenId(rat.Msg.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID), zap.String("contractAddr", c.Address)).Warn("Failed to get token id for transfer")
			continue
		}

		nft, err := i.nftRepo.GetNft(c.Address, tokenId)
		if err != nil {
			pendingRequest := i.elastic.GetRequest(entity.CreateNftSlug(tokenId, c.Address))
			if pendingRequest != nil {
				nft = pendingRequest.Entity.(entity.NFT)
			} else {
				time.Sleep(2 * time.Second)
				nft, err = i.nftRepo.GetNft(c.Address, tokenId)
				if err != nil {
					zap.L().With(zap.Error(err), zap.String("contract", c.Address), zap.Uint64("tokenId", tokenId)).Error("Failed to find nft in index")

					continue
				}
			}
		}

		previousOwner := nft.Owner

		newOwner, err := rat.Msg.Params.GetParam("recipient")
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get new owner")
			return err
		}

		nft.Owner = newOwner.Value.Primitive.(string)
		newOwnerBech32, _ := bech32.ToBech32Address(nft.Owner)
		nft.OwnerBech32 = newOwnerBech32

		zap.L().With(
			zap.Uint64("blockNum", tx.BlockNum),
			zap.String("symbol", nft.Symbol),
			zap.Uint64("tokenId", nft.TokenId),
			zap.String("from", previousOwner),
			zap.String("to", nft.Owner),
		).Info("Transfer NFT")
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nft)
	}

	zap.L().With(zap.Int("count", len(rats))).Info("Index nft transfers")

	return nil
}
