package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_cache"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
)

var (
	ErrNftNotFound = errors.New("nft not found")
)

type NftRepository interface {
	GetNft(contract string, tokenId uint64) (entity.NFT, error)
	GetNfts(contract string) ([]entity.NFT, int64, error)
}

type nftRepository struct {
	elastic elastic_cache.Index
}

func NewNftRepository(elastic elastic_cache.Index) NftRepository {
	return nftRepository{elastic}
}

func (r nftRepository) GetNft(contract string, tokenId uint64) (entity.NFT, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", contract),
		elastic.NewTermQuery("tokenId", tokenId),
	)

	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.NftIndex.Get()).
		Query(query).
		Size(1))

	return r.findOne(result, err)
}

func (r nftRepository) GetNfts(contract string) ([]entity.NFT, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", contract),
	)

	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.NftIndex.Get()).
		Query(query).
		TrackTotalHits(true).
		Size(100))

	return r.findMany(result, err)
}

func (r nftRepository) findOne(results *elastic.SearchResult, err error) (entity.NFT, error) {
	if err != nil {
		return entity.NFT{}, err
	}

	if len(results.Hits.Hits) == 0 {
		return entity.NFT{}, ErrNftNotFound
	}

	var nft entity.NFT
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &nft)

	return nft, err
}

func (r nftRepository) findMany(results *elastic.SearchResult, err error) ([]entity.NFT, int64, error) {
	nfts := make([]entity.NFT, 0)

	if err != nil {
		return nfts, 0, err
	}

	for _, hit := range results.Hits.Hits {
		var nft entity.NFT
		if err := json.Unmarshal(hit.Source, &nft); err == nil {
			nfts = append(nfts, nft)
		}
	}

	return nfts, results.TotalHits(), nil
}
