package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"go.uber.org/zap"
	"os"
	"strconv"
)

var container *dic.Container

func main() {
	initialize()

	args := os.Args[1:]
	blockNum, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to get block num arg")
	}

	txs, err := container.GetTxIndexer().Index(uint64(blockNum), 1)
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to index")
	}

	for _, tx := range txs {
		zap.S().Infof("Indexed %s", tx.ID)
	}

	container.GetElastic().Persist()
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
	zap.L().Info("Import Block")
}
