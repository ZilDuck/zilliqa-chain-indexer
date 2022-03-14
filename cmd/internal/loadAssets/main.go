package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"go.uber.org/zap"
	"os"
	"sync"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	nftRepo := container.GetNftRepo()
	metadataIndexer := container.GetMetadataIndexer()
	elastic := container.GetElastic()

	contractAddr := os.Args[1]
	onlyMissing := len(os.Args) >= 3 && os.Args[2] == "true"
	force := len(os.Args) >= 4 && os.Args[3] == "true"

	size := 10
	page := 1

	for {
		var nfts []entity.Nft
		var err error
		if contractAddr == "all" {
			nfts, _, err = nftRepo.GetAllNfts(size, page)
		} else {
			nfts, _, err = nftRepo.GetNfts(contractAddr, size, page)
		}

		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts")
			panic(err)
		}

		zap.S().Infof("Page: %d", page)

		if len(nfts) == 0 {
			break
		}

		var wg sync.WaitGroup
		for _, nft := range nfts {
			wg.Add(1)

			go func(nft entity.Nft) {
				defer wg.Done()
				if onlyMissing == true && (nft.Metadata.UriEmpty() || nft.MediaUri != "") {
					if nft.Metadata.UriEmpty() {
						nft.Metadata = factory.GetMetadata(nft)
						elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.NftMetadata)
						elastic.BatchPersist()
					}
					return
				}
				if onlyMissing == false || nft.MediaUri == "" {
					if err = metadataIndexer.RefreshMetadata(nft.Contract, nft.TokenId); err != nil {
						zap.L().With(zap.Error(err)).Error("Failed to refresh metadata")
						return
					}
				}
				if onlyMissing == false || nft.MediaUri == "" {
					if err = metadataIndexer.RefreshAsset(nft.Contract, nft.TokenId, force); err != nil {
						zap.L().With(zap.Error(err)).Error("Failed to refresh asset")
						return
					}
				}
				elastic.BatchPersist()
			}(nft)
		}
		wg.Wait()
		elastic.Persist()

		page++
	}
	elastic.Persist()
}