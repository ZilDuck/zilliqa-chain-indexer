package di

import (
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/indexer"
	"github.com/dantudor/zil-indexer/internal/service/contract"
	"github.com/dantudor/zil-indexer/internal/service/nft"
	"github.com/dantudor/zil-indexer/internal/service/transaction"
	"github.com/dantudor/zil-indexer/internal/service/zilliqa"
	"github.com/patrickmn/go-cache"
	"github.com/sarulabs/dingo/v3"
	"go.uber.org/zap"
	"time"
)

var Definitions = []dingo.Def{
	{
		Name: "elastic",
		Build: func() (elastic_cache.Index, error) {
			elastic, err := elastic_cache.New()
			if err != nil {
				zap.L().With(zap.Error(err)).Fatal("Failed to start ES")
			}

			return elastic, nil
		},
	},
	{
		Name: "cache",
		Build: func() (*cache.Cache, error) {
			return cache.New(5*time.Minute, 10*time.Minute), nil
		},
	},
	{
		Name: "zilliqa",
		Build: func() (zilliqa.Service, error) {
			p := zilliqa.NewProvider(config.Get().Zilliqa.Url)
			return zilliqa.NewZilliqaService(p), nil
		},
	},
	{
		Name: "indexer",
		Build: func(
			elastic elastic_cache.Index,
			txIndexer transaction.Indexer,
			txService transaction.Service,
			contractIndexer contract.Indexer,
			nftIndexer nft.Indexer,
			rewinder indexer.Rewinder,
		) (indexer.Indexer, error) {
			return indexer.NewIndexer(config.Get().BulkIndexSize, elastic, txIndexer, txService, contractIndexer, nftIndexer, rewinder), nil
		},
	},
	{
		Name: "rewinder",
		Build: func(
			elastic elastic_cache.Index,
			txRewinder transaction.Rewinder,
			txService transaction.Service,
			txRepo transaction.Repository,
			contractRewinder nft.Rewinder,
			nftRewinder nft.Rewinder,
		) (indexer.Rewinder, error) {
			return indexer.NewRewinder(elastic, txRewinder, txService, txRepo, contractRewinder, nftRewinder), nil
		},
	},
	{
		Name: "tx.indexer",
		Build: func(
			zilliqa zilliqa.Service,
			elastic elastic_cache.Index,
			transactionFactory transaction.Factory,
			repository transaction.Repository,
			service transaction.Service,
		) (transaction.Indexer, error) {
			return transaction.NewIndexer(zilliqa, elastic, transactionFactory, repository, service), nil
		},
	},
	{
		Name: "tx.rewinder",
		Build: func(elastic elastic_cache.Index) (transaction.Rewinder, error) {
			return transaction.NewRewinder(elastic), nil
		},
	},
	{
		Name: "tx.repo",
		Build: func(elastic elastic_cache.Index) (transaction.Repository, error) {
			return transaction.NewRepo(elastic), nil
		},
	},
	{
		Name: "tx.service",
		Build: func(repository transaction.Repository, cache *cache.Cache) (transaction.Service, error) {
			return transaction.NewService(repository, cache), nil
		},
	},
	{
		Name: "tx.factory",
		Build: func(zilliqa zilliqa.Service) (transaction.Factory, error) {
			return transaction.NewTransactionFactory(zilliqa), nil
		},
	},
	{
		Name: "contract.indexer",
		Build: func(
			elastic elastic_cache.Index,
			factory contract.Factory,
			txRepo transaction.Repository,
		) (contract.Indexer, error) {
			return contract.NewIndexer(elastic, factory, txRepo), nil
		},
	},
	{
		Name: "contract.rewinder",
		Build: func(elastic elastic_cache.Index) (contract.Rewinder, error) {
			return contract.NewRewinder(elastic), nil
		},
	},
	{
		Name: "contract.factory",
		Build: func(zilliqa zilliqa.Service) (contract.Factory, error) {
			return contract.NewFactory(zilliqa), nil
		},
	},
	{
		Name: "contract.repo",
		Build: func(
			elastic elastic_cache.Index,
		) (contract.Repository, error) {
			return contract.NewRepo(elastic), nil
		},
	},
	{
		Name: "nft.indexer",
		Build: func(elastic elastic_cache.Index, contractRepo contract.Repository, nftRepo nft.Repository, txRepo transaction.Repository) (nft.Indexer, error) {
			return nft.NewIndexer(elastic, contractRepo, nftRepo, txRepo), nil
		},
	},
	{
		Name: "nft.rewinder",
		Build: func(elastic elastic_cache.Index) (nft.Rewinder, error) {
			return nft.NewRewinder(elastic), nil
		},
	},
	{
		Name: "nft.repo",
		Build: func(elastic elastic_cache.Index) (nft.Repository, error) {
			return nft.NewRepo(elastic), nil
		},
	},
}
