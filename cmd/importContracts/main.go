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

	//c, err := container.GetContractRepo().GetContractByAddress("0x20f29c1931ab65b22b92840c355f6e02ae92659e")
	//if err != nil {
	//	panic(err)
	//}
	//
	//if err := container.GetNftIndexer().IndexContract(c); err != nil {
	//	panic(err)
	//}
	//container.GetElastic().Persist()

	if err := container.GetContractIndexer().BulkIndex(uint64(0)); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
	container.GetElastic().InstallMappings()
}
