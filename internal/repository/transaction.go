package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/entity"
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
	GetContractCreationTx(contractAddress string) (entity.Transaction, error)
	GetContractTxs(contractAddress string, size, from int) ([]entity.Transaction, int64, error)

	GetContractCreationTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)
	GetContractExecutionTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)
	GetContractExecutionsWithTransition(contractAddr string, transitionName entity.TRANSITION, size, from int) ([]entity.Transaction, int64, error)

	GetMintersForZrc1Contract(contract string) ([]string, error)
}

type transactionRepository struct {
	elastic elastic_cache.Index
}

func NewTransactionRepository(elastic elastic_cache.Index) TransactionRepository {
	return transactionRepository{elastic}
}

func (r transactionRepository) GetTx(txId string) (entity.Transaction, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(elastic.NewTermQuery("ID", txId)).
		Do(context.Background())

	return r.findOne(results, err)
}

func (r transactionRepository) GetContractCreationTx(contractAddress string) (entity.Transaction, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractAddress.keyword", contractAddress),
		elastic.NewTermQuery("ContractCreation", true),
	)

	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Size(1).
		Do(context.Background())

	return r.findOne(result, err)
}

func (r transactionRepository) GetContractTxs(contractAddress string, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractAddress.keyword", contractAddress),
		elastic.NewTermQuery("ContractExecution", true),
	)

	from := size*page - size

	zap.L().With(
		zap.String("contractAddress", contractAddress),
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetContractTxs")

	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(result, err)
}

func (r transactionRepository) GetBestBlockNum() (uint64, error) {
	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Sort("BlockNum", false).
		Size(1).
		Do(context.Background())
	if err != nil || result == nil {
		time.Sleep(5 * time.Second)
		return 0, err
	}

	if len(result.Hits.Hits) == 0 {
		return 0, ErrBestBlockNumFound
	}

	var tx *entity.Transaction
	hit := result.Hits.Hits[0]
	if err = json.Unmarshal(hit.Source, &tx); err != nil {
		return 0, err
	}

	return tx.BlockNum, nil
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

	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

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

	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(result, err)
}

func (r transactionRepository) GetMintTxsForContract(contract string, size, from int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewNestedQuery("Receipt", elastic.NewTermQuery("Receipt.success", true)),
		elastic.NewNestedQuery("Receipt.transitions", elastic.NewTermQuery("Receipt.transitions.addr.keyword", contract)),
		elastic.NewNestedQuery("Receipt.transitions.msg", elastic.NewTermQuery("Receipt.transitions.msg._tag.keyword", "Mint")),
	)
	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(result, err)
}

func (r transactionRepository) GetContractExecutionsWithTransition(contractAddr string, transition entity.TRANSITION, size, from int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewNestedQuery("Receipt", elastic.NewTermQuery("Receipt.success", true)),
		elastic.NewNestedQuery("Receipt.transitions", elastic.NewTermQuery("Receipt.transitions.addr.keyword", contractAddr)),
		elastic.NewNestedQuery("Receipt.transitions.msg", elastic.NewTermQuery("Receipt.transitions.msg._tag.keyword", transition)),
	)

	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(result, err)
}

func (r transactionRepository) GetMintersForZrc1Contract(contract string) (minters []string, err error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractAddress.keyword", contract),
		elastic.NewTermQuery("Data._tag.keyword", "ConfigureMinter"),
		elastic.NewNestedQuery("Receipt", elastic.NewTermQuery("Receipt.success", true)),
		elastic.NewNestedQuery("Receipt.event_logs", elastic.NewTermQuery("Receipt.event_logs._eventname.keyword", "AddMinterSuccess")),
	)
	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(1000).
		Do(context.Background())

	txs, _, err := r.findMany(result, err)
	if err != nil {
		return
	}

	for _, tx := range txs {
		for _, eventLog := range tx.GetEventLogs("AddMinterSuccess") {
			if eventLog.Params.HasParam("minter", "ByStr20") {
				if minter, err := eventLog.Params.GetParam("minter"); err == nil {
					minters = append(minters, minter.Value.Primitive.(string))
				}
			}
		}
	}

	return
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
