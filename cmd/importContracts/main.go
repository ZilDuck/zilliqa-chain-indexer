package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/dev"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"time"
)

var container *dic.Container

func main() {
	defer sentry.Flush(2 * time.Second)

	initialize()

	tx, _ := container.GetTxRepo().GetTx("496fb5a7292ae6e8cede9e4a3203c5c828d9fc5f49d253f04e3e2c0b0f274d5b")
	dev.DD(tx)

	//tx, _ := container.GetTxRepo().GetTx("e0e5716b47f5fdfeaeb523b0c8ff0691d1113feb1914e4d73e1f79f1e658d583")
	//container.GetContractIndexer().Index([]entity.Transaction{tx})
	//container.GetElastic().Persist()
	//panic(nil)

	//c, _ := container.GetContractRepo().GetContractByAddress("0x52ae7144748c55d3951f74585cd84bb65439ca98")
	//container.GetZrc6Indexer().IndexContract(c)
	//dev.DD(c)
	//panic(nil)

	if err := container.GetContractIndexer().BulkIndex(uint64(3500000)); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
	container.GetElastic().InstallMappings()
}
