package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"go.uber.org/zap"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()

	txRepo := container.GetTxRepo()
	elastic := container.GetElastic()

	bestBlock, _ := container.GetTxRepo().GetBestBlockNum()
	zap.S().Infof("Transaction index best block: %d", bestBlock)

	var from uint64 = 1
	var size uint64 = 100

	for {
		txs, _ := container.GetTxIndexer().CreateTransactions(from, size)
		for _, tx := range txs {
			if _, err := txRepo.GetTx(tx.ID); err != nil {
				zap.L().With(zap.Error(err), zap.String("txID", tx.ID)).Error("Tx Error")
				elastic.AddIndexRequest(elastic_search.TransactionIndex.Get(), tx, elastic_search.TransactionCreate)
			}
		}
		elastic.BatchPersist()

		from = from + size

		if from >= bestBlock {
			break
		}
	}
	elastic.Persist()
}
