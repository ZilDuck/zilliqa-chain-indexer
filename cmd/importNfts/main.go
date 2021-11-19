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

	container.GetElastic().InstallMappings()

	size := 100
	page := 1

	for {
		contracts, _, err := container.GetContractRepo().GetAllZrc1Contracts(size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts")
			panic(err)
		}
		if len(contracts) == 0 {
			break
		}
		for _, c := range contracts {
			err := container.GetNftIndexer().IndexContract(c)
			if err != nil {
				zap.S().Errorf("Failed to index NFTs for contract %s", c.Address)
			}
		}
		container.GetElastic().BatchPersist()
		page++
	}
	container.GetElastic().Persist()
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
}
