package contract

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"github.com/olivere/elastic/v7"
)

var (
	ErrContractNotFound = errors.New("contract not found")
)

type Repository interface {
	GetAllZrc1Contracts(size, from int) ([]zil.Contract, int64, error)
	GetContractByAddress(contractAddr string) (zil.Contract, error)
	GetContractByAddressBech32(contractAddr string) (zil.Contract, error)
	GetContractByMinterFallbackToAddress(contractAddr string) (zil.Contract, error)
}

type repository struct {
	elastic elastic_cache.Index
}

func NewRepo(elastic elastic_cache.Index) Repository {
	return repository{elastic}
}

func (r repository) GetAllZrc1Contracts(size, from int) ([]zil.Contract, int64, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("zrc1", true)).
		Sort("blockNum", true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(results, err)
}

func (r repository) GetContractByAddress(contractAddr string) (zil.Contract, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("address.keyword", contractAddr)).
		Do(context.Background())

	return r.findOne(results, err)
}

func (r repository) GetContractByAddressBech32(contractAddr string) (zil.Contract, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("addressBech32.keyword", contractAddr)).
		Do(context.Background())

	return r.findOne(results, err)
}

func (r repository) GetContractByMinterFallbackToAddress(contractAddr string) (zil.Contract, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("minters.keyword", contractAddr)).
		Do(context.Background())

	contract, err := r.findOne(results, err)
	if err != nil {
		if err == ErrContractNotFound {
			contract, err = r.GetContractByAddress(contractAddr)
		}
	}

	return contract, err
}

func (r repository) findOne(results *elastic.SearchResult, err error) (zil.Contract, error) {
	if err != nil {
		return zil.Contract{}, err
	}

	if len(results.Hits.Hits) == 0 {
		return zil.Contract{}, ErrContractNotFound
	}

	var contract zil.Contract
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &contract)

	return contract, err
}

func (r repository) findMany(results *elastic.SearchResult, err error) ([]zil.Contract, int64, error) {
	contracts := make([]zil.Contract, 0)

	if err != nil {
		return contracts, 0, err
	}

	for _, hit := range results.Hits.Hits {
		var contract zil.Contract
		if err := json.Unmarshal(hit.Source, &contract); err == nil {
			contracts = append(contracts, contract)
		}
	}

	return contracts, results.TotalHits(), nil
}
