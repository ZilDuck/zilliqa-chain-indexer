package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"time"
)

var container *dic.Container

func main() {
	defer sentry.Flush(2 * time.Second)

	initialize()

	if err := container.GetContractIndexer().BulkIndex(config.Get().FirstBlockNum); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
	container.GetElastic().InstallMappings()
}
