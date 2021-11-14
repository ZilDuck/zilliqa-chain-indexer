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

	container.GetElastic().InstallMappings()

	if err := container.GetNftIndexer().BulkIndex(); err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to bulk index NFTs")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
}
