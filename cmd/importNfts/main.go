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

	container.GetElastic().InstallMappings()

	c, err := container.GetContractRepo().GetContractByAddress("0xd2b54e791930dd7d06ea51f3c2a6cf2c00f165ea")
	if err != nil {
		panic(err)
	}

	size := 10
	page := 1
	for {
		nfts, _, err := container.GetNftRepo().GetNfts(c.Address, size, page)
		if err != nil {
			panic(err)
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
					container.GetZrc6Factory().FetchImage(&n)
					container.GetElastic().AddUpdateRequest(elastic_cache.NftIndex.Get(), n, elastic_cache.Zrc6SetBaseUri)
				} else {
					zap.L().With(zap.Uint64("tokenId", n.TokenId)).Info("Media already discovered")
				}
			}(nft)
		}
		wg.Wait()

		container.GetElastic().Persist()
		page++
	}

	panic(nil)
	//
	//size := 100
	//page := 1
	//
	//for {
	//	contracts, total, err := container.GetContractRepo().GetAllNftContracts(size, page)
	//	if err != nil {
	//		zap.L().With(zap.Error(err)).Error("Failed to get contracts")
	//		panic(err)
	//	}
	//	if page == 1 {
	//		zap.S().Infof("Found %d ZRC1 contracts", total)
	//	}
	//	if len(contracts) == 0 {
	//		break
	//	}
	//	for _, c := range contracts {
	//		if err := container.GetZrc1Indexer().IndexContract(c); err != nil {
	//			zap.S().Errorf("Failed to index ZRC1 NFTs for contract %s", c.Address)
	//		}
	//		if err := container.GetZrc6Indexer().IndexContract(c, true); err != nil {
	//			zap.S().Errorf("Failed to index ZRC6 NFTs for contract %s", c.Address)
	//		}
	//	}
	//	container.GetElastic().BatchPersist()
	//	page++
	//}
	//container.GetElastic().Persist()
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
}
