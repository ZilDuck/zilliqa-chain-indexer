package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"sync"
)

func main() {
	config.Init()
	container, _ := dic.NewContainer()
	metadataService := container.GetMetadataService()
	elastic := container.GetElastic()

	size := 20
	page := 1
	for {
		nfts, _, err := container.GetNftRepo().GetAllNfts(size, page)
		if err != nil || len(nfts) == 0 {
			break
		}

		var wg sync.WaitGroup

		for _, nft := range nfts {
			wg.Add(1)
			go func () {
				defer wg.Done()

				if nft.Metadata.Status == entity.MetadataPending {
					properties, err := metadataService.FetchMetadata(nft)
					if err != nil {
						nft.Metadata.Error = err.Error()
						nft.Metadata.Status = entity.MetadataFailure
						nft.Metadata.Attempts = 1
						zap.L().With(zap.String("contractAddr", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Error("NFT metadata failure")
					} else {
						zap.L().With(zap.String("contractAddr", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("NFT metadata indexed")
						nft.Metadata.Error = ""
						nft.Metadata.Properties = properties
						nft.Metadata.Status = entity.MetadataSuccess
					}
					elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.NftMetadata)
				}
			}()
			wg.Wait()
			elastic.BatchPersist()
		}

		page++
	}
	elastic.Persist()
}