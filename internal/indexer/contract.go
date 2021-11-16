package indexer

import (
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/entity"
	"github.com/dantudor/zil-indexer/internal/factory"
	"github.com/dantudor/zil-indexer/internal/repository"
	"go.uber.org/zap"
)

type ContractIndexer interface {
	Index(txs []entity.Transaction) ([]entity.Contract, error)
	BulkIndex(fromBlockNum uint64) error
}

type contractIndexer struct {
	elastic elastic_cache.Index
	factory factory.ContractFactory
	txRepo  repository.TransactionRepository
}

func NewContractIndexer(
	elastic elastic_cache.Index,
	factory factory.ContractFactory,
	txRepo repository.TransactionRepository,
) ContractIndexer {
	return contractIndexer{elastic, factory, txRepo}
}

func (i contractIndexer) Index(txs []entity.Transaction) ([]entity.Contract, error) {
	contracts := make([]entity.Contract, 0)
	for _, tx := range txs {
		if tx.Receipt.Success == false || tx.IsContractCreation == false {
			continue
		}

		contract, err := i.factory.CreateContractFromTx(tx)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("failed to create contract")
			return nil, err
		}

		i.elastic.AddIndexRequest(elastic_cache.ContractIndex.Get(), contract)
		contracts = append(contracts, contract)
	}

	zap.L().With(zap.Int("count", len(contracts))).Info("Index contracts")

	return contracts, nil
}

func (i contractIndexer) BulkIndex(fromBlockNum uint64) error {
	zap.L().With(zap.Uint64("from", fromBlockNum)).Info("Bulk index contracts")
	size := 100
	page := 1

	for {
		txs, _, err := i.txRepo.GetContractCreationTxs(fromBlockNum, size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contract txs")
			return err
		}

		if len(txs) == 0 {
			zap.L().Info("No more contract creation txs found")
			break
		}

		for _, tx := range txs {
			contract, err := i.factory.CreateContractFromTx(tx)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to create contract txs")
				break
			}
			if !contract.ZRC1 {
				continue
			}
			zap.L().With(
				zap.Uint64("blockNum", contract.BlockNum),
				zap.String("name", contract.Name),
				zap.String("address", contract.Address),
				zap.Bool("zrc1", contract.ZRC1),
			).Info("Index contract")

			//if contract.ZRC1 {
			minters, err := i.txRepo.GetMintersForZrc1Contract(contract.Address)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("txId", tx.ID)).Error("Failed to get contract minters")
				return err
			}
			contract.Minters = minters
			if contract.Minters != nil {
				zap.S().Infof("Adding minters to: %s", contract.Address)
			}
			//}

			i.elastic.AddIndexRequest(elastic_cache.ContractIndex.Get(), contract)
		}

		i.elastic.BatchPersist()

		page++
	}

	i.elastic.Persist()

	return nil
}
