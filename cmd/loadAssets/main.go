package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"go.uber.org/zap"
	"os"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	nftRepo := container.GetNftRepo()
	zrc6Indexer := container.GetZrc6Indexer()

	contractAddr := os.Args[1]

	onlyMissing := len(os.Args) == 3 && os.Args[2] == "true"

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

		for _, nft := range nfts {
			if onlyMissing == true && nft.Metadata != nil && nft.MediaUri != "" {
				continue
			}
			zrc6Indexer.TriggerMetadataRefresh(nft)
		}

		page++
	}
}