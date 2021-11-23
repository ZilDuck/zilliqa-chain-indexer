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
		contracts, total, err := container.GetContractRepo().GetAllNftContracts(size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts")
			panic(err)
		}
		if page == 1 {
			zap.S().Infof("Found %d ZRC1 contracts", total)
		}
		if len(contracts) == 0 {
			break
		}
		for _, c := range contracts {
			if err := container.GetZrc1Indexer().IndexContract(c); err != nil {
				zap.S().Errorf("Failed to index ZRC1 NFTs for contract %s", c.Address)
			}
			if err := container.GetZrc6Indexer().IndexContract(c); err != nil {
				zap.S().Errorf("Failed to index ZRC6 NFTs for contract %s", c.Address)
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
