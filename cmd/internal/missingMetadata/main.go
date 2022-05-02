package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"go.uber.org/zap"
)

func main() {
	config.Init("missingMetadata")

	container, _ := dic.NewContainer()
	metadataIndexer := container.GetMetadataIndexer()

	page := 1
	size := 100

	for {
		nfts, _, err := container.GetNftRepo().GetAllNfts(size, page)
		if err != nil {
			zap.L().Fatal(err.Error())
		}

		if len(nfts) == 0 {
			zap.L().Info("Finished")
			break
		}
		zap.S().Infof("Page %d", page)

		for _, nft := range nfts {
			if nft.Metadata != nil {
				if nft.Metadata.Status == "failure" {
					zap.S().Infof("Refresh metadata for token %d", nft.TokenId)
					metadataIndexer.RefreshMetadata(nft.Contract, nft.TokenId)
				}
			}

		}
		container.GetElastic().Persist()
		page ++
	}
}

