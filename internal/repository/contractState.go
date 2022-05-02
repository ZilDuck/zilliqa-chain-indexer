package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
	"strconv"
)

var (
	ErrContractStateNotFound = errors.New("contract state not found")
)

type ContractStateRepository interface {
	GetStateByAddress(contractAddr string) (*entity.ContractState, error)
	GetRoyaltyFeeBps(contractAddr string) (uint, error)
}

type contractStateRepository struct {
	elastic elastic_search.Index
}

func NewContractStateRepository(elastic elastic_search.Index) ContractStateRepository {
	return contractStateRepository{elastic}
}

func (r contractStateRepository) GetStateByAddress(contractAddr string) (*entity.ContractState, error) {
	pendingRequest := r.elastic.GetRequest(entity.CreateStateSlug(contractAddr))
	if pendingRequest != nil {
		pendingState := pendingRequest.Entity.(entity.ContractState)
		return &pendingState, nil
	}

	results, err := search(r.elastic.GetClient().
		Search(elastic_search.ContractStateIndex.Get()).
		Query(elastic.NewTermQuery("address.keyword", contractAddr)))

	state, err := r.findOne(results, err)

	return state, err
}

func (r contractStateRepository)  GetRoyaltyFeeBps(contractAddr string) (uint, error) {
	state, err := r.GetStateByAddress(contractAddr)
	if err != nil {
		return 0, err
	}

	royaltyFeeBps, exists := state.GetElement("royalty_fee_bps")
	if !exists {
		return 0, nil
	}

	fee, err := strconv.Atoi(royaltyFeeBps)
	if err != nil {
		return 0, err
	}

	return uint(fee), nil
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
