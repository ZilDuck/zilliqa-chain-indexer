package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/dev"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"time"
)

var container *dic.Container

func main() {
	defer sentry.Flush(2 * time.Second)

	initialize()

	tx, err := container.GetTxRepo().GetTx("e0e5716b47f5fdfeaeb523b0c8ff0691d1113feb1914e4d73e1f79f1e658d583")
	if err != nil {
		panic(err)
	}

	contacts, err := container.GetContractIndexer().Index([]entity.Transaction{tx})
	if err != nil {
		panic(err)
	}

	c := contacts[0]

	txs, total, err := container.GetTxRepo().GetContractTxs(c.Address, 100, 1)
	if err != nil {
		panic(err)
	}

	dev.Dump(total)
	for _, tx := range txs {
		zap.L().Info(tx.ID)
		dev.Dump(tx)
	}

	container.GetNftIndexer().Index(txs)

	//if err := container.GetContractIndexer().BulkIndex(uint64(0)); err != nil {
	//	zap.L().Fatal("Failed to bulk index contracts")
	//}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
}
