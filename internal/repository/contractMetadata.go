package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
)

var (
	ErrContractMetadataNotFound = errors.New("contract metadata not found")
)

type ContractMetadataRepository interface {
	GetMetadataByAddress(contractAddr string) (*entity.ContractMetadata, error)
}

type contractMetadataRepository struct {
	elastic elastic_search.Index
}

func NewContractMetadataRepository(elastic elastic_search.Index) ContractMetadataRepository {
	return contractMetadataRepository{elastic}
}

func (r contractMetadataRepository) GetMetadataByAddress(contractAddr string) (*entity.ContractMetadata, error) {
	results, err := search(r.elastic.GetClient().
		Search(elastic_search.ContractMetadataIndex.Get()).
		Query(elastic.NewTermQuery("contract.keyword", contractAddr)))

	state, err := r.findOne(results, err)

	return state, err
}

func (r contractMetadataRepository) findOne(results *elastic.SearchResult, err error) (*entity.ContractMetadata, error) {
	if err != nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, ErrContractMetadataNotFound
	}

	var md entity.ContractMetadata
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &md)

	return &md, nil
}
