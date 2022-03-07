package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

var (
	ErrContractStateNotFound = errors.New("contract state not found")
)

type ContractStateRepository interface {
	GetState(contractAddr string) (*entity.ContractState, error)
}

type contractStateRepository struct {
	elastic elastic_search.Index
}

func NewContractStateRepository(elastic elastic_search.Index) ContractStateRepository {
	return contractStateRepository{elastic}
}

func (r contractStateRepository) GetState(contractAddr string) (*entity.ContractState, error) {
	results, err := search(r.elastic.GetClient().
		Search(elastic_search.ContractStateIndex.Get()).
		Query(elastic.NewTermQuery("contract.keyword", contractAddr)))

	c, err := r.findOne(results, err)
	if err != nil && errors.Is(err, ErrContractStateNotFound) {
		zap.S().Warnf("%s: %s", err.Error(), contractAddr)
	}

	return c, err
}

func (r contractStateRepository) findOne(results *elastic.SearchResult, err error) (*entity.ContractState, error) {
	if err != nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, ErrContractStateNotFound
	}

	var state entity.ContractState
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &state)

	return &state, nil
}