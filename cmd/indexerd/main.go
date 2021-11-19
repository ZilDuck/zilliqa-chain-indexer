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
	initialize()
	defer sentry.Flush(2 * time.Second)

	container.GetDaemon().Execute()
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
	zap.L().Info("Indexer Started")
}
