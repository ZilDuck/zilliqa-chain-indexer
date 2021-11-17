package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

var (
	ErrContractNotFound = errors.New("contract not found")
)

type ContractRepository interface {
	GetAllZrc1Contracts(size, page int) ([]entity.Contract, int64, error)
	GetContractByAddress(contractAddr string) (entity.Contract, error)
	GetContractByAddressBech32(contractAddr string) (entity.Contract, error)
	GetContractByMinterFallbackToAddress(contractAddr string) (entity.Contract, error)
}

type contractRepository struct {
	elastic elastic_cache.Index
}

func NewContractRepository(elastic elastic_cache.Index) ContractRepository {
	return contractRepository{elastic}
}

func (r contractRepository) GetAllZrc1Contracts(size, page int) ([]entity.Contract, int64, error) {
	from := size*page - size

	zap.L().With(
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetAllZrc1Contracts")

	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("zrc1", true)).
		Sort("blockNum", true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(results, err)
}

func (r contractRepository) GetContractByAddress(contractAddr string) (entity.Contract, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("address.keyword", contractAddr)).
		Do(context.Background())

	return r.findOne(results, err)
}

func (r contractRepository) GetContractByAddressBech32(contractAddr string) (entity.Contract, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.ContractIndex.Get()).
		Query(elastic.NewTermQuery("addressBech32.keyword", contractAddr)).
		Do(context.Background())

	return r.findOne(results, err)
}

func (r contractRepository) GetContractByMinterFallbackToAddress(contractAddr string) (entity.Contract, error) {
	zap.S().Debugf("GetContractByMinterFallbackToAddress: %s", contractAddr)

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

func (r contractRepository) findOne(results *elastic.SearchResult, err error) (entity.Contract, error) {
	if err != nil {
		return entity.Contract{}, err
	}

	if len(results.Hits.Hits) == 0 {
		return entity.Contract{}, ErrContractNotFound
	}

	var contract entity.Contract
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &contract)

	return contract, err
}

func (r contractRepository) findMany(results *elastic.SearchResult, err error) ([]entity.Contract, int64, error) {
	contracts := make([]entity.Contract, 0)

	if err != nil {
		return contracts, 0, err
	}

	for _, hit := range results.Hits.Hits {
		var contract entity.Contract
		if err := json.Unmarshal(hit.Source, &contract); err == nil {
			contracts = append(contracts, contract)
		}
	}

	return contracts, results.TotalHits(), nil
}
