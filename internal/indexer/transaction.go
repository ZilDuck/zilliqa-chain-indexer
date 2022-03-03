package indexer

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"go.uber.org/zap"
)

type TransactionIndexer interface {
	Index(height, size uint64) ([]entity.Transaction, error)
	CreateTransactions(height uint64, size uint64) ([]entity.Transaction, error)
}

type transactionIndexer struct {
	zilliqa   zilliqa.Service
	elastic   elastic_search.Index
	txFactory factory.TransactionFactory
	txRepo    repository.TransactionRepository
}

func NewTransactionIndexer(
	zilliqa zilliqa.Service,
	elastic elastic_search.Index,
	factory factory.TransactionFactory,
	txRepo repository.TransactionRepository,
) TransactionIndexer {
	return transactionIndexer{zilliqa, elastic, factory, txRepo}
}

func (i transactionIndexer) Index(height, size uint64) ([]entity.Transaction, error) {
	txs, err := i.CreateTransactions(height, size)
	if err != nil {
		return nil, err
	}

	for _, tx := range txs {
		i.elastic.AddIndexRequest(elastic_search.TransactionIndex.Get(), tx, elastic_search.TransactionCreate)
	}

	return txs, nil
}

func (i transactionIndexer) CreateTransactions(height uint64, size uint64) ([]entity.Transaction, error) {
	coreTxGroups, err := i.zilliqa.GetTxnBodiesForTxBlocks(height, size)
	if err != nil {
		return nil, err
	}

	txs := make([]entity.Transaction, 0)
	contractCreationTxs := make([]entity.Transaction, 0)
	for blockNum, coreTxs := range coreTxGroups {
		for _, coreTx := range coreTxs {
			if coreTx.Receipt.Success == false {
				continue
			}

			tx := i.txFactory.CreateTransaction(coreTx, blockNum)
			if tx.IsContractCreation {
				contractCreationTxs = append(contractCreationTxs, tx)
			}

			if !tx.IsContractCreation && !tx.IsContractExecution {
				continue
			}

			txs = append(txs, tx)
		}
	}

	contractAddrs, err := i.zilliqa.GetContractAddressFromTransactionIDs(getTxIds(contractCreationTxs))
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to get contract addresses")
		return nil, err
	}

	for idx, _ := range txs {
		if contractAddr, ok := contractAddrs[txs[idx].ID]; ok {
			if contractAddr != "" {
				txs[idx].ContractAddress = fmt.Sprintf("0x%s", contractAddr)
			}
		}
	}

	zap.L().With(zap.Int("count", len(txs)), zap.Uint64("height", height)).Info("Index txs")

	return txs, nil
}

func getTxIds(txs []entity.Transaction) []string {
	var txIds []string
	for _, tx := range txs {
		txIds = append(txIds, tx.ID)
	}
	return txIds
}
