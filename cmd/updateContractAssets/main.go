package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_cache"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"sync"
	"time"
)

var container *dic.Container

func main() {
	defer sentry.Flush(2 * time.Second)

	initialize()

	elastic := container.GetElastic()
	elastic.InstallMappings()

	nftRepo := container.GetNftRepo()
	zrc6Factory := container.GetZrc6Factory()

	c, _ := container.GetContractRepo().GetContractByAddress("0xd2b54e791930dd7d06ea51f3c2a6cf2c00f165ea")

	size := 10
	page := 1

	for {
		nfts, _, err := nftRepo.GetNfts(c.Address, size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get NFTs")
			break
		}
		if len(nfts) == 0 {
			break
		}

		var wg sync.WaitGroup
		for _, nft := range nfts {
			wg.Add(1)

			go func(n entity.Nft) {
				defer wg.Done()
				if n.MediaUri == "" {
					if err := zrc6Factory.FetchImage(&n); err != nil {
						zap.L().With(zap.Error(err)).Error("Failed to fetch image")
						return
					}
					elastic.AddIndexRequest(elastic_cache.NftIndex.Get(), n, elastic_cache.Zrc6Mint)
				}
			}(nft)
		}
		wg.Wait()

		elastic.BatchPersist()
		page++
	}

	elastic.Persist()
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
}
