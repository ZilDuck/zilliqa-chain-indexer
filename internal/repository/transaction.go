package repository

import (
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
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

	GetTx(txId string) (*entity.Transaction, error)
	GetMissingTxs(txIds []string) (map[string]struct{}, error)

	GetContractCreationTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)
	GetContractExecutionTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)

	GetContractCreationForContract(contractAddr string) (*entity.Transaction, error)
	GetContractExecutionsByContract(c entity.Contract, size, page int) ([]entity.Transaction, int64, error)
	GetContractExecutionsByContractFrom(c entity.Contract, fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)

	GetNftMarketplaceExecutionTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error)
}

type transactionRepository struct {
	elastic elastic_search.Index
}

func NewTransactionRepository(elastic elastic_search.Index) TransactionRepository {
	return transactionRepository{elastic}
}

func (r transactionRepository) GetBestBlockNum() (uint64, error) {
	result, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
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
		Search(elastic_search.TransactionIndex.Get()).
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

func (r transactionRepository) GetTx(txId string) (*entity.Transaction, error) {
	zap.L().Debug("TransactionRepository::GetTx: "+txId)
	results, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
		Query(elastic.NewTermQuery("ID", txId)))

	return r.findOne(results, err)
}

func (r transactionRepository) GetMissingTxs(txIds []string) (map[string]struct{}, error) {
	zap.L().Debug("TransactionRepository::GetMissingTxs")
	values := make([]interface{}, len(txIds))
	for i, v := range txIds {
		values[i] = v
	}

	results, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
		Query(elastic.NewTermsQuery("ID.keyword", values...)).
		Aggregation("txId", elastic.NewTermsAggregation().Field("ID.keyword").Size(len(txIds))))
	if err != nil {
		return nil, err
	}

	missingTxIds := map[string]struct{}{}
	for _, tx := range txIds {
		missingTxIds[tx] = struct{}{}
	}

	if txId, found := results.Aggregations.Terms("txId"); found {
		for _, txIdBucket := range txId.Buckets {
			delete(missingTxIds, txIdBucket.Key.(string))
		}
	}

	return missingTxIds, nil
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
		Search(elastic_search.TransactionIndex.Get()).
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
		Search(elastic_search.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from))

	return r.findMany(result, err)
}

func (r transactionRepository) GetContractCreationForContract(contractAddr string) (*entity.Transaction, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractCreation", true),
		elastic.NewTermQuery("ContractAddress.keyword", contractAddr),
	)

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		Size(1).
		TrackTotalHits(true))

	return r.findOne(result, err)
}

func (r transactionRepository) GetContractExecutionsByContract(c entity.Contract, size, page int) ([]entity.Transaction, int64, error) {
	zap.L().Info("Get contract executions for " + c.Address)
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
	).Should(
		elastic.NewNestedQuery("Receipt.event_logs", elastic.NewTermQuery("Receipt.event_logs.address.keyword", c.Address)),
		elastic.NewTermQuery("ContractAddress.keyword", c.Address),
	).MinimumShouldMatch("1")

	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		Size(size).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r transactionRepository) GetContractExecutionsByContractFrom(c entity.Contract, fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewRangeQuery("BlockNum").Gte(fromBlockNum),
		elastic.NewNestedQuery("Receipt.event_logs", elastic.NewTermQuery("Receipt.event_logs.address.keyword", c.Address)),
	)
	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		Size(size).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r transactionRepository) GetNftMarketplaceExecutionTxs(fromBlockNum uint64, size, page int) ([]entity.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractExecution", true),
		elastic.NewRangeQuery("BlockNum").Gte(fromBlockNum),
		elastic.NewBoolQuery().Should(
			zilkRoadExecutionTxs(),
			arkyExecutionTxs(),
			okimotoExecutionTxs(),
			mintableExecutionTxs(),
		).MinimumNumberShouldMatch(1),
	)

	from := size*page - size

	zap.L().With(
		zap.Int("size", size),
		zap.Int("page", page),
		zap.Int("from", from),
	).Info("GetNftMarketplaceExecutionTxs")

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.TransactionIndex.Get()).
		Query(query).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from))

	return r.findMany(result, err)
}

func (r transactionRepository) findOne(results *elastic.SearchResult, err error) (*entity.Transaction, error) {
	if err != nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, errors.New("no transaction found")
	}

	var tx entity.Transaction
	if err = json.Unmarshal(results.Hits.Hits[0].Source, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
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

func zilkRoadExecutionTxs() *elastic.BoolQuery {
	return elastic.NewBoolQuery().Should(
		elastic.NewNestedQuery("Receipt.event_logs", elastic.NewBoolQuery().Should(
			elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpZilkroadListingEvent),
			elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpZilkroadDelistingEvent),
			elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpZilkroadSaleEvent),
		).MinimumNumberShouldMatch(1)),
	).MinimumNumberShouldMatch(1)
}

func arkyExecutionTxs() *elastic.BoolQuery {
	return elastic.NewBoolQuery().Should(
		elastic.NewNestedQuery("Receipt.event_logs", elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpArkySaleEvent)),
	).MinimumNumberShouldMatch(1)
}

func okimotoExecutionTxs() *elastic.BoolQuery {
	return elastic.NewBoolQuery().Should(
		elastic.NewBoolQuery().Must(
			elastic.NewMatchPhraseQuery("Data._tag.keyword", "ConfigurePrice"),
			elastic.NewMatchPhraseQuery("ContractAddress.keyword", entity.OkimotoMarketplaceAddress),
		),
		elastic.NewBoolQuery().Must(
			elastic.NewNestedQuery("Receipt.event_logs", elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpOkiDelistingEvent)),
			elastic.NewMatchPhraseQuery("Data._tag.keyword", "WithdrawalToken"),
		),
		elastic.NewBoolQuery().Must(
			elastic.NewNestedQuery("Receipt.event_logs", elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpOkiSaleEvent)),
			elastic.NewMatchPhraseQuery("Data._tag.keyword", "Buy"),
		),
	).MinimumNumberShouldMatch(1)
}

func mintableExecutionTxs() *elastic.BoolQuery {
	return elastic.NewBoolQuery().Should(
		elastic.NewNestedQuery("Receipt.event_logs", elastic.NewBoolQuery().Should(
			elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpMintableListingEvent),
			elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpMintableDelistingEvent),
			elastic.NewMatchPhraseQuery("Receipt.event_logs._eventname.keyword", entity.MpMintableSaleEvent),
		).MinimumNumberShouldMatch(1)),
	).MinimumNumberShouldMatch(1)
}