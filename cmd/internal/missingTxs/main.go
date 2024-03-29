package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"go.uber.org/zap"
	"os"
	"strconv"
)

func main() {
	config.Init("missingTx")

	container, _ := dic.NewContainer()

	txRepo := container.GetTxRepo()
	elastic := container.GetElastic()

	bestBlock, _ := container.GetTxRepo().GetBestBlockNum()
	zap.S().Infof("Transaction index best block: %d", bestBlock)


	var from uint64 = 0
	if len(os.Args) == 2 {
		if i, err := strconv.ParseInt(os.Args[1], 10, 64); err == nil {
			from = uint64(i)
		}
	}

	var size uint64 = 100

	for {
		txs, _ := container.GetTxIndexer().CreateTransactions(from, size)
		if len(txs) > 0 {
			txIds := make([]string, len(txs))
			for idx, tx := range txs {
				txIds[idx] = tx.ID
			}

			missingTxs, err := txRepo.GetMissingTxs(txIds)
			if err != nil {
				panic(err)
			}

			for _, tx := range txs {
				if _, ok := missingTxs[tx.ID]; ok {
					zap.L().With(zap.Error(err), zap.String("txID", tx.ID)).Info("Missing Tx")
					elastic.AddIndexRequest(elastic_search.TransactionIndex.Get(), tx, elastic_search.TransactionCreate)
				}
			}
			elastic.Persist()
		}

		from = from + size

		if from >= bestBlock {
			break
		}
	}
	elastic.Persist()
}
