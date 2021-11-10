package transaction

import (
	"context"
	"encoding/json"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"github.com/olivere/elastic/v7"
)

type Repository interface {
	GetBestBlockNum() (uint64, error)
	GetTx(txId string) (zil.Transaction, error)
	GetContractCreationTx(contractAddress string) (zil.Transaction, error)
	GetContractTxs(contractAddress string, size, from int) ([]zil.Transaction, int64, error)

	GetContractCreationTxs(size, from int) ([]zil.Transaction, int64, error)
	GetContractExecutionTxs(size, from int) ([]zil.Transaction, int64, error)
	GetContractExecutionsWithTransition(contractAddr string, transitionName zil.TRANSITION, size, from int) ([]zil.Transaction, int64, error)

	//GetMintTxsForContract(contract string, size, from int) ([]zil.Transaction, int64, error)
	GetMintersForZrc1Contract(contract string) ([]string, error)
}

type repository struct {
	elastic elastic_cache.Index
}

func NewRepo(elastic elastic_cache.Index) Repository {
	return repository{elastic}
}

func (r repository) GetTx(txId string) (zil.Transaction, error) {
	results, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(elastic.NewTermQuery("ID", txId)).
		Do(context.Background())

	return r.findOne(results, err)
}

func (r repository) GetContractCreationTx(contractAddress string) (zil.Transaction, error) {
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

func (r repository) GetContractTxs(contractAddress string, size, from int) ([]zil.Transaction, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("ContractAddress.keyword", contractAddress),
		elastic.NewTermQuery("ContractExecution", true),
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

func (r repository) GetBestBlockNum() (uint64, error) {
	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Sort("BlockNum", false).
		Size(1).
		Do(context.Background())
	if err != nil || result == nil {
		return 0, err
	}

	if len(result.Hits.Hits) == 0 {
		return 0, ErrTxBlockTransactionNotFound
	}

	var tx *zil.Transaction
	hit := result.Hits.Hits[0]
	if err = json.Unmarshal(hit.Source, &tx); err != nil {
		return 0, err
	}

	return tx.BlockNum, nil
}

func (r repository) GetContractCreationTxs(size, from int) ([]zil.Transaction, int64, error) {
	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(elastic.NewTermQuery("ContractCreation", true)).
		Sort("BlockNum", true).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(result, err)
}

func (r repository) GetContractExecutionTxs(size, from int) ([]zil.Transaction, int64, error) {
	result, err := r.elastic.GetClient().
		Search(elastic_cache.TransactionIndex.Get()).
		Query(elastic.NewTermQuery("ContractExecution", true)).
		Sort("BlockNum", false).
		TrackTotalHits(true).
		Size(size).
		From(from).
		Do(context.Background())

	return r.findMany(result, err)
}

func (r repository) GetMintTxsForContract(contract string, size, from int) ([]zil.Transaction, int64, error) {
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

func (r repository) GetContractExecutionsWithTransition(contractAddr string, transition zil.TRANSITION, size, from int) ([]zil.Transaction, int64, error) {
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

func (r repository) GetMintersForZrc1Contract(contract string) (minters []string, err error) {
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

func (r repository) findOne(results *elastic.SearchResult, err error) (zil.Transaction, error) {
	if err != nil {
		return zil.Transaction{}, err
	}

	if len(results.Hits.Hits) == 0 {
		return zil.Transaction{}, ErrTxBlockTransactionNotFound
	}

	var tx zil.Transaction
	err = json.Unmarshal(results.Hits.Hits[0].Source, &tx)

	return tx, err
}

func (r repository) findMany(results *elastic.SearchResult, err error) ([]zil.Transaction, int64, error) {
	txs := make([]zil.Transaction, 0)

	if err != nil {
		return txs, 0, err
	}

	for _, hit := range results.Hits.Hits {
		var tx zil.Transaction
		if err := json.Unmarshal(hit.Source, &tx); err == nil {
			txs = append(txs, tx)
		}
	}

	return txs, results.TotalHits(), nil
}
