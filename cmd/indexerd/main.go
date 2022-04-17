package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"go.uber.org/zap"
)

var container *dic.Container

func main() {
	initialize()

	container.GetDaemon().Execute()
}

func initialize() {
	config.Init("indexer")
	container, _ = dic.NewContainer()
	zap.L().Info("Indexer Started")
}
