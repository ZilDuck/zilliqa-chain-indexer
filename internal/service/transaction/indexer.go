package transaction

import (
	"fmt"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/service/zilliqa"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"go.uber.org/zap"
)

type Indexer interface {
	Index(height, size uint64) ([]zil.Transaction, error)
	CreateTransactions(height uint64, size uint64) ([]zil.Transaction, error)
}

type indexer struct {
	zilliqaService zilliqa.Service
	elastic        elastic_cache.Index
	factory        Factory
	repository     Repository
	service        Service
}

func NewIndexer(
	zilliqaService zilliqa.Service,
	elastic elastic_cache.Index,
	factory Factory,
	repository Repository,
	service Service,
) Indexer {
	return indexer{
		zilliqaService,
		elastic,
		factory,
		repository,
		service,
	}
}

func (i indexer) Index(height, size uint64) ([]zil.Transaction, error) {
	txs, err := i.CreateTransactions(height, size)
	if err != nil {
		return nil, err
	}

	i.service.SetLastBlockNumIndexed(height + size - 1)

	return txs, nil
}

func (i indexer) CreateTransactions(height uint64, size uint64) ([]zil.Transaction, error) {
	coreTxGroups, err := i.zilliqaService.GetTxnBodiesForTxBlocks(height, size)
	if err != nil {
		return nil, err
	}

	txs := make([]zil.Transaction, 0)
	contractCreationTxs := make([]zil.Transaction, 0)
	for blockNum, coreTxs := range coreTxGroups {
		for _, coreTx := range coreTxs {
			if coreTx.Receipt.Success == false {
				continue
			}

			tx := i.factory.CreateTransaction(coreTx, blockNum)
			if tx.IsContractCreation {
				contractCreationTxs = append(contractCreationTxs, tx)
			}

			if !tx.IsContractCreation && !tx.IsContractExecution {
				continue
			}

			i.elastic.AddIndexRequest(elastic_cache.TransactionIndex.Get(), tx)
			txs = append(txs, tx)
		}
	}

	contractAddrs, err := i.zilliqaService.GetContractAddressFromTransactionIDs(getTxIds(contractCreationTxs))
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to get contract addresses")
		return nil, err
	}

	for idx, _ := range txs {
		if contractAddr, ok := contractAddrs[txs[idx].ID]; ok {
			txs[idx].ContractAddress = fmt.Sprintf("0x%s", contractAddr)
			txs[idx].ContractAddressBech32 = getBech32Address(contractAddr)
			i.elastic.AddUpdateRequest(elastic_cache.TransactionIndex.Get(), txs[idx])
		}
	}

	zap.L().With(zap.Int("count", len(txs)), zap.Uint64("height", height)).Info("Index txs")

	return txs, nil
}

func getTxIds(txs []zil.Transaction) []string {
	var txIds []string
	for _, tx := range txs {
		txIds = append(txIds, tx.ID)
	}
	return txIds
}
