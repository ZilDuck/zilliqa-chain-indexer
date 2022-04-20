package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
)

var (
	ErrNftActionNotFound = errors.New("nft action not found")
)

type NftActionRepository interface {
	GetNftOwnerBeforeBlockNum(nft entity.Nft, blockNum uint64) (string, error)
}

type nftActionRepository struct {
	elastic elastic_search.Index
}

func NewNftActionRepository(elastic elastic_search.Index) NftActionRepository {
	return nftActionRepository{elastic}
}

func (r nftActionRepository) GetNftOwnerBeforeBlockNum(nft entity.Nft, blockNum uint64) (string, error) {
	//pendingRequest := r.elastic.GetRequest(entity.CreateNftActionSlug(nft.TokenId, nft.Contract))
	//if pendingRequest != nil {
	//	pendingState := pendingRequest.Entity.(entity.ContractState)
	//	return &pendingState, nil
	//}

	query := elastic.NewBoolQuery().Must(
		elastic.NewRangeQuery("blockNum").Lt(blockNum),
		elastic.NewTermQuery("contract.keyword", nft.Contract),
		elastic.NewTermQuery("tokenId", nft.TokenId),
		elastic.NewTermsQuery("action.keyword", "mint", "transfer"),
	)

	results, err := search(r.elastic.GetClient().
		Search(elastic_search.NftActionIndex.Get()).
		Query(query).
		Size(1))

	action, err := r.findOne(results, err)
	if err != nil {
		return "", err
	}

	return action.To, nil
}

func (r nftActionRepository) findOne(results *elastic.SearchResult, err error) (*entity.NftAction, error) {
	if err != nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, ErrNftActionNotFound
	}

	var action entity.NftAction
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &action)

	return &action, nil
}
