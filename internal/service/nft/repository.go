package nft

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"github.com/olivere/elastic/v7"
)

var (
	ErrNftNotFound = errors.New("nft not found")
)

type Repository interface {
	GetNft(contract string, tokenId uint64) (zil.NFT, error)
}

type repository struct {
	elastic elastic_cache.Index
}

func NewRepo(elastic elastic_cache.Index) Repository {
	return repository{elastic}
}

func (r repository) GetNft(contract string, tokenId uint64) (zil.NFT, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", contract),
		elastic.NewTermQuery("tokenId", tokenId),
	)

	result, err := r.elastic.GetClient().
		Search(elastic_cache.NftIndex.Get()).
		Query(query).
		Size(1).
		Do(context.Background())

	return r.findOne(result, err)
}

func (r repository) findOne(results *elastic.SearchResult, err error) (zil.NFT, error) {
	if err != nil {
		return zil.NFT{}, err
	}

	if len(results.Hits.Hits) == 0 {
		return zil.NFT{}, ErrNftNotFound
	}

	var nft zil.NFT
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &nft)

	return nft, err
}
