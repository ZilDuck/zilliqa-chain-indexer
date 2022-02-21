package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"os"
	"sync"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	nftRepo := container.GetNftRepo()
	zrc6Indexer := container.GetZrc6Indexer()
	elastic := container.GetElastic()

	contractAddr := os.Args[1]

	size := 10
	page := 1

	for {
		nfts, _, err := nftRepo.GetNfts(contractAddr, size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts")
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
				if n.Zrc6 == true {
					err := zrc6Indexer.RefreshAsset(n.Contract, n.TokenId)
					if err != nil {
						zap.L().With(zap.Error(err)).Error("Failed to fetch zrc6 asset")
						return
					}
					elastic.BatchPersist()
				}
			}(nft)
		}
		wg.Wait()
		elastic.Persist()

		page++
	}
}