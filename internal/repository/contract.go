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
	ErrContractNotFound = errors.New("contract not found")
)

type ContractRepository interface {
	GetAllContracts(size, page int) ([]entity.Contract, int64, error)
	GetAllNftContracts(size, page int) ([]entity.Contract, int64, error)
	GetContractByAddress(contractAddr string) (*entity.Contract, error)
}

type contractRepository struct {
	elastic elastic_search.Index
}

func NewContractRepository(elastic elastic_search.Index) ContractRepository {
	return contractRepository{elastic}
}

func (r contractRepository) GetAllContracts(size, page int) ([]entity.Contract, int64, error) {
	from := size*page - size

	zap.L().With(
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetAllContracts")

	results, err := search(r.elastic.GetClient().
		Search(elastic_search.ContractIndex.Get()).
		Sort("blockNum", false).
		Size(size).
		From(from))

	return r.findMany(results, err)
}

func (r contractRepository) GetAllNftContracts(size, page int) ([]entity.Contract, int64, error) {
	from := size*page - size

	zap.L().With(
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetAllNftContracts")

	query := elastic.NewBoolQuery().Should(
		elastic.NewTermQuery("zrc1", true),
		elastic.NewTermQuery("zrc6", true),
	).MinimumShouldMatch("1")

	results, err := search(r.elastic.GetClient().
		Search(elastic_search.ContractIndex.Get()).
		Query(query).
		Sort("blockNum", true).
		Size(size).
		From(from))

	return r.findMany(results, err)
}

func (r contractRepository) GetContractByAddress(contractAddr string) (*entity.Contract, error) {
	results, err := search(r.elastic.GetClient().
		Search(elastic_search.ContractIndex.Get()).
		Query(elastic.NewTermQuery("address.keyword", contractAddr)))

	c, err := r.findOne(results, err)
	if err != nil && errors.Is(err, ErrContractNotFound) {
		zap.S().Warnf("%s: %s", err.Error(), contractAddr)
	}

	return c, err
}

func (r contractRepository) findOne(results *elastic.SearchResult, err error) (*entity.Contract, error) {
	if err != nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, ErrContractNotFound
	}

	var contract entity.Contract
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &contract)

	return &contract, nil
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
