package main

import (
	"github.com/dantudor/zil-indexer/generated/dic"
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/sarulabs/dingo/v3"
	"go.uber.org/zap"
)

var container *dic.Container

func main() {
	initialize()

	if err := container.GetContractIndexer().BulkIndex(); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
}
