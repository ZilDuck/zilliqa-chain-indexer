package indexer

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type ContractIndexer interface {
	Index(txs []entity.Transaction) error
	BulkIndex(fromBlockNum uint64) error
}

type contractIndexer struct {
	elastic      elastic_search.Index
	factory      factory.ContractFactory
	txRepo       repository.TransactionRepository
	contractRepo repository.ContractRepository
	nftRepo      repository.NftRepository
}

func NewContractIndexer(
	elastic elastic_search.Index,
	factory factory.ContractFactory,
	txRepo repository.TransactionRepository,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
) ContractIndexer {
	return contractIndexer{elastic, factory, txRepo, contractRepo, nftRepo}
}

func (i contractIndexer) Index(txs []entity.Transaction) error {
	for _, tx := range txs {
		if tx.Receipt.Success == false {
			continue
		}

		if tx.IsContractCreation {
			c, err := i.factory.CreateContractFromTx(tx)
			if err == nil {
				zap.L().With(
					zap.Uint64("blockNum", c.BlockNum),
					zap.String("name", c.Name),
					zap.String("address", c.Address),
					zap.Bool("zrc1", c.ZRC1),
					zap.Bool("zrc6", c.ZRC6),
				).Info("Index contract")

				i.elastic.AddIndexRequest(elastic_search.ContractIndex.Get(), c, elastic_search.ContractCreate)
			}
		}
	}

	return nil
}

func (i contractIndexer) BulkIndex(fromBlockNum uint64) error {
	zap.L().With(zap.Uint64("from", fromBlockNum)).Info("Bulk index contracts")
	size := 50
	page := 1

	for {
		txs, _, err := i.txRepo.GetContractCreationTxs(fromBlockNum, size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contract txs")
			return err
		}
		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			c, err := i.factory.CreateContractFromTx(tx)
			if err != nil {
				continue
			}

			zap.L().With(
				zap.Uint64("blockNum", c.BlockNum),
				zap.String("name", c.Name),
				zap.String("address", c.Address),
				zap.Bool("zrc1", c.ZRC1),
				zap.Bool("zrc6", c.ZRC6),
			).Info("Index contract")

			i.elastic.AddIndexRequest(elastic_search.ContractIndex.Get(), c, elastic_search.ContractCreate)
		}

		i.elastic.Persist()

		page++
	}

	i.elastic.Persist()

	return nil
}
