package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_cache"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"time"
)

var (
	ErrBestBlockNumFound = errors.New("best block num not found")
)

type TransactionRepository interface {
	GetBestBlockNum() (uint64, error)
	GetTx(txId string) (entity.Transaction, error)

	GetContractCreationTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)
	GetContractExecutionTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)

	GetContractExecutionsByContract(c entity.Contract, size, page int) ([]entity.Transaction, int64, error)
	GetContractExecutionsByContractFrom(c entity.Contract, fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)
}

type transactionRepository struct {
	elastic elastic_cache.Index
}

func NewTransactionRepository(elastic elastic_cache.Index) TransactionRepository {
	return transactionRepository{elastic}
}

func (r transactionRepository) GetBestBlockNum() (uint64, error) {
	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Size(1))

	if err != nil {
		time.Sleep(5 * time.Second)
		return 0, err
	}

	if result == nil || len(result.Hits.Hits) == 0 {
		zap.L().Info("Best block num not found giving 0")
		return 0, ErrBestBlockNumFound
	}

	result, err = search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Sort("BlockNum", false).
		Size(1))
	if err != nil {
		time.Sleep(5 * time.Second)
		return 0, err
	}

	var tx *entity.Transaction
	hit := result.Hits.Hits[0]
	if err = json.Unmarshal(hit.Source, &tx); err != nil {
		return 0, err
	}

	return tx.BlockNum, nil
}

func (r transactionRepository) GetTx(txId string) (entity.Transaction, error) {
	results, err := search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(elastic.NewTermQuery("ID", txId)))

	return r.findOne(results, err)
}

func (r transactionRepository) GetContractCreationTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractCreation", true),
		elastic.NewRangeQuery("BlockNum").Gte(fromBlockNum),
	)

	from := size*page - size

	zap.L().With(
		zap.Uint64("blockNum", fromBlockNum),
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetContractCreationTxs")

	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from))

	return r.findMany(result, err)
}

func (r transactionRepository) GetContractExecutionTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewRangeQuery("BlockNum").Gte(fromBlockNum),
	)

	from := size*page - size

	zap.L().With(
		zap.Uint64("blockNum", fromBlockNum),
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetContractExecutionTxs")

	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from))

	return r.findMany(result, err)
}

func (r transactionRepository) GetContractExecutionsByContract(c entity.Contract, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewNestedQuery("Receipt.transitions", elastic.NewTermQuery("Receipt.transitions.addr.keyword", c.Address)),
	)
	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		Size(size).
		From(from).
		TrackTotalHits(true).
		Size(size))

	return r.findMany(result, err)
}

func (r transactionRepository) GetContractExecutionsByContractFrom(c entity.Contract, fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewRangeQuery("BlockNum").Gte(fromBlockNum),
		elastic.NewNestedQuery("Receipt.transitions", elastic.NewTermQuery("Receipt.transitions.addr.keyword", c.Address)),
	)
	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		Size(size).
		From(from).
		TrackTotalHits(true).
		Size(size))

	return r.findMany(result, err)
}

func (r transactionRepository) findOne(results *elastic.SearchResult, err error) (entity.Transaction, error) {
	if err != nil {
		return entity.Transaction{}, err
	}

	if len(results.Hits.Hits) == 0 {
		return entity.Transaction{}, errors.New("no transaction found")
	}

	var tx entity.Transaction
	err = json.Unmarshal(results.Hits.Hits[0].Source, &tx)

	return tx, err
}

func (r transactionRepository) findMany(results *elastic.SearchResult, err error) ([]entity.Transaction, int64, error) {
	txs := make([]entity.Transaction, 0)

	if err != nil {
		return txs, 0, err
	}

	for _, hit := range results.Hits.Hits {
		var tx entity.Transaction
		if err := json.Unmarshal(hit.Source, &tx); err == nil {
			txs = append(txs, tx)
		}
	}

	return txs, results.TotalHits(), nil
}
