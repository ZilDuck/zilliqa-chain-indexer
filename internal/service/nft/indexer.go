package nft

import (
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/service/contract"
	"github.com/dantudor/zil-indexer/internal/service/transaction"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"go.uber.org/zap"
	"log"
	"time"
)

var (
	defaultSize = 100
)

type Indexer interface {
	Index(txs []zil.Transaction) error
	BulkIndex() error

	IndexTxMints(tx zil.Transaction, c zil.Contract) error
	IndexTxDuckRegenerations(tx zil.Transaction, c zil.Contract) error
	IndexTxTransfers(tx zil.Transaction, c zil.Contract) error

	IndexContract(c zil.Contract) error
	IndexContractMints(c zil.Contract) error
	IndexContractDuckRegenerations(c zil.Contract) error
	IndexContractTransfers(c zil.Contract) error
}

type indexer struct {
	elastic      elastic_cache.Index
	contractRepo contract.Repository
	nftRepo      Repository
	txRepo       transaction.Repository
}

func NewIndexer(elastic elastic_cache.Index, contractRepo contract.Repository, nftRepo Repository, txRepo transaction.Repository) Indexer {
	return indexer{elastic, contractRepo, nftRepo, txRepo}
}

func (i indexer) Index(txs []zil.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		for _, tx := range txs {
			c, err := i.contractRepo.GetContractByMinterFallbackToAddress(tx.ContractAddress)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("contractAddr", tx.ContractAddress)).Error("Failed to find contract")
				continue
			}

			if c.ZRC1 == false {
				continue
			}

			if err := i.IndexTxMints(tx, c); err != nil {
				return err
			}

			if err := i.IndexTxDuckRegenerations(tx, c); err != nil {
				return err
			}

			if err := i.IndexTxTransfers(tx, c); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i indexer) BulkIndex() error {
	size := defaultSize
	from := 0

	for {
		contracts, _, err := i.contractRepo.GetAllZrc1Contracts(size, from)
		if err != nil {
			zap.L().With(zap.Error(err)).Fatal("Failed to get contract")
		}

		if len(contracts) == 0 {
			break
		}

		for _, c := range contracts {
			err := i.IndexContract(c)
			if err != nil {
				zap.S().Errorf("Failed to index NFTs for contract %s", c.Address)
			}
		}

		i.elastic.BatchPersist()

		zap.S().Warnf("Moving to page: %d", from)
		from = from + size - 1
	}

	i.elastic.Persist()

	return nil
}

func (i indexer) IndexContract(c zil.Contract) error {
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

func (i indexer) IndexTxMints(tx zil.Transaction, c zil.Contract) error {
	nfts, err := CreateNftsFromMintingTx(tx, c)
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

func (i indexer) IndexContractMints(c zil.Contract) (err error) {
	zap.L().With(zap.String("contractAddr", c.Address)).Info("Index Contract Mints")

	indexContractMints := func(contractAddr string) error {
		size := defaultSize
		from := 0
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

func (i indexer) IndexTxDuckRegenerations(tx zil.Transaction, c zil.Contract) error {
	transitions := tx.GetTransition(zil.TransitionRegenerateDuck)
	for _, transition := range transitions {
		if !transition.Msg.Params.HasParam("token_id", "Uint256") {
			continue
		}
		tokenId, _ := GetTokenId(transition.Msg.Params)

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
		zap.L().With(zap.String("symbol", nft.Symbol), zap.Uint64("tokenId", nft.TokenId)).Info("Regenerate NFD")
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nft)
	}
	zap.L().With(zap.Int("count", len(transitions))).Info("Index nft duck regenerations")

	return nil
}

func (i indexer) IndexContractDuckRegenerations(c zil.Contract) error {
	if !c.HasTransition(zil.TransitionRegenerateDuck) {
		return nil
	}

	zap.L().Info("Index Duck Regenerations")

	indexDuckRegenerations := func(contractAddr string) error {
		size := defaultSize
		from := 0

		for {
			txs, _, err := i.txRepo.GetContractExecutionsWithTransition(contractAddr, zil.TransitionRegenerateDuck, size, from)
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

func (i indexer) IndexTxTransfers(tx zil.Transaction, c zil.Contract) error {
	err := i.handleTransfersForTx(c, tx)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to handle transfers")
	}
	return err
}

func (i indexer) IndexContractTransfers(c zil.Contract) error {
	zap.L().With(zap.String("contractAddr", c.Address)).Info("Index Contract Transfers")

	size := defaultSize
	from := 0

	for {
		txs, _, err := i.txRepo.GetContractExecutionsWithTransition(c.Address, zil.TransitionRecipientAcceptTransfer, size, from)
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

func (i indexer) handleTransfersForTx(contract zil.Contract, tx zil.Transaction) error {
	rats := tx.GetTransition(zil.TransitionRecipientAcceptTransfer)
	for _, rat := range rats {
		tokenId, err := GetTokenId(rat.Msg.Params)
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Warn("Failed to get token id for transfer")
			continue
		}

		nft, err := i.nftRepo.GetNft(contract.Address, tokenId)
		if err != nil {
			pendingRequest := i.elastic.GetRequest(zil.CreateNftSlug(tokenId, contract.Address))
			if pendingRequest != nil {
				nft = pendingRequest.Entity.(zil.NFT)
			} else {
				time.Sleep(2 * time.Second)
				nft, err = i.nftRepo.GetNft(contract.Address, tokenId)
				if err != nil {
					zap.L().With(zap.Error(err), zap.String("contract", contract.Address), zap.Uint64("tokenId", tokenId)).Error("Failed to find nft in index")

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

		zap.L().With(zap.String("symbol", nft.Symbol), zap.Uint64("tokenId", nft.TokenId),
			zap.String("from", previousOwner), zap.String("to", nft.Owner)).Info("Transfer NFT")
		i.elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), nft)
	}
	zap.L().With(zap.Int("count", len(rats))).Info("Index nft transfers")

	return nil
}
