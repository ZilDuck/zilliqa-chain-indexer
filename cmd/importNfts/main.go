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

	size := 100
	from := 0

	for {
		contracts, _, err := container.GetContractRepo().GetAllZrc1Contracts(size, from)
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
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
}
