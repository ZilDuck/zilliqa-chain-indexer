package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"sync"
)

func main() {
	config.Init()
	container, _ := dic.NewContainer()
	metadataIndexer := container.GetMetadataIndexer()
	elastic := container.GetElastic()

	size := 100
	page := 1
	for {
		nfts, total, err := container.GetNftRepo().GetPendingMetadata(size, page)
		if err != nil || len(nfts) == 0 {
			break
		}
		if page == 1 {
			zap.S().Infof("Found %d NFTS", total)
		}

		var wg sync.WaitGroup

		zap.S().Infof("Processing page %d", page)
		for _, nft := range nfts {
			wg.Add(1)
			go func () {
				defer wg.Done()

				if nft.Metadata.Status == entity.MetadataPending {
					//metadataIndexer.RefreshMetadata(nft.Contract, nft.TokenId)
					metadataIndexer.TriggerMetadataRefresh(nft)
				}
			}()
			wg.Wait()
			elastic.Persist()
		}

		page++
	}
	elastic.Persist()
}