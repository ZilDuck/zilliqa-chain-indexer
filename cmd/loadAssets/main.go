package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"go.uber.org/zap"
	"os"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	nftRepo := container.GetNftRepo()
	metadataIndexer := container.GetMetadataIndexer()
	elastic := container.GetElastic()

	contractAddr := os.Args[1]
	onlyMissing := len(os.Args) == 3 && os.Args[2] == "true"

	size := 1000
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

		for _, nft := range nfts {
			if onlyMissing == true && (nft.Metadata.UriEmpty() || nft.MediaUri != "") {
				if nft.Metadata.UriEmpty() {
					nft.Metadata = factory.GetMetadata(nft)
					elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), nft, elastic_search.NftMetadata)
					elastic.BatchPersist()
				}
				continue
			}
			metadataIndexer.TriggerMetadataRefresh(nft)
		}

		page++
	}
	elastic.Persist()
}