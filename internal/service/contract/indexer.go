package contract

import (
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/service/transaction"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"go.uber.org/zap"
)

type Indexer interface {
	Index(txs []zil.Transaction) ([]zil.Contract, error)
	BulkIndex() error
}

type indexer struct {
	elastic elastic_cache.Index
	factory Factory
	txRepo  transaction.Repository
}

func NewIndexer(elastic elastic_cache.Index, factory Factory, txRepo transaction.Repository) Indexer {
	return indexer{elastic, factory, txRepo}
}

func (i indexer) Index(txs []zil.Transaction) ([]zil.Contract, error) {
	contracts := make([]zil.Contract, 0)
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

func (i indexer) BulkIndex() error {
	size := 100
	from := 0

	for {
		contractTxs, _, err := i.txRepo.GetContractCreationTxs(size, from)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contract txs")
			return err
		}

		if len(contractTxs) == 0 {
			break
		}

		for _, contractTx := range contractTxs {
			contract, err := i.factory.CreateContractFromTx(contractTx)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("txId", contractTx.ID)).Error("Failed to create contract txs")
				break
			}
			if contract.Name == "Resolver" || contract.Name == "QuizBot" {
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
				zap.L().With(zap.Error(err), zap.String("txId", contractTx.ID)).Error("Failed to get contract minters")
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

		from = from + size - 1
		zap.S().Infof("Moving to page: %d", from)
	}

	i.elastic.Persist()

	return nil
}
