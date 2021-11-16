package main

import (
	"github.com/dantudor/zil-indexer/generated/dic"
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/sarulabs/dingo/v3"
	"go.uber.org/zap"
	"time"
)

var container *dic.Container

func main() {
	defer sentry.Flush(2 * time.Second)

	initialize()

	if err := container.GetContractIndexer().BulkIndex(uint64(0)); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
}
