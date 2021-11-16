package di

import (
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/dantudor/zil-indexer/internal/daemon"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/factory"
	"github.com/dantudor/zil-indexer/internal/indexer"
	"github.com/dantudor/zil-indexer/internal/repository"
	"github.com/dantudor/zil-indexer/internal/zilliqa"
	"github.com/patrickmn/go-cache"
	"github.com/sarulabs/dingo/v4"
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
		Name: "daemon",
		Build: func(
			elastic elastic_cache.Index,
			indexer indexer.Indexer,
			zilliqa zilliqa.Service,
			txRepo repository.TransactionRepository,
			contractIndexer indexer.ContractIndexer,
			nftIndexer indexer.NftIndexer,
		) (*daemon.Daemon, error) {
			return daemon.NewDaemon(elastic, indexer, zilliqa, txRepo, contractIndexer, nftIndexer), nil
		},
	},
	{
		Name: "indexer",
		Build: func(
			elastic elastic_cache.Index,
			txIndexer indexer.TransactionIndexer,
			contractIndexer indexer.ContractIndexer,
			nftIndexer indexer.NftIndexer,
			txRepo repository.TransactionRepository,
			cache *cache.Cache,
		) (indexer.Indexer, error) {
			return indexer.NewIndexer(config.Get().BulkIndexSize, elastic, txIndexer, contractIndexer, nftIndexer, txRepo, cache), nil
		},
	},
	{
		Name: "tx.indexer",
		Build: func(
			zilliqa zilliqa.Service,
			elastic elastic_cache.Index,
			transactionFactory factory.TransactionFactory,
			txRepo repository.TransactionRepository,
		) (indexer.TransactionIndexer, error) {
			return indexer.NewTransactionIndexer(zilliqa, elastic, transactionFactory, txRepo), nil
		},
	},
	{
		Name: "contract.indexer",
		Build: func(
			elastic elastic_cache.Index,
			factory factory.ContractFactory,
			txRepo repository.TransactionRepository,
		) (indexer.ContractIndexer, error) {
			return indexer.NewContractIndexer(elastic, factory, txRepo), nil
		},
	},
	{
		Name: "nft.indexer",
		Build: func(
			elastic elastic_cache.Index,
			contractRepo repository.ContractRepository,
			nftRepo repository.NftRepository,
			txRepo repository.TransactionRepository,
		) (indexer.NftIndexer, error) {
			return indexer.NewNftIndexer(elastic, contractRepo, nftRepo, txRepo), nil
		},
	},
	{
		Name: "tx.repo",
		Build: func(elastic elastic_cache.Index) (repository.TransactionRepository, error) {
			return repository.NewTransactionRepository(elastic), nil
		},
	},
	{
		Name: "contract.repo",
		Build: func(elastic elastic_cache.Index) (repository.ContractRepository, error) {
			return repository.NewContractRepository(elastic), nil
		},
	},
	{
		Name: "nft.repo",
		Build: func(elastic elastic_cache.Index) (repository.NftRepository, error) {
			return repository.NewNftRepository(elastic), nil
		},
	},
	{
		Name: "tx.factory",
		Build: func(zilliqa zilliqa.Service) (factory.TransactionFactory, error) {
			return factory.NewTransactionFactory(zilliqa), nil
		},
	},
	{
		Name: "contract.factory",
		Build: func(zilliqa zilliqa.Service) (factory.ContractFactory, error) {
			return factory.NewContractFactory(zilliqa), nil
		},
	},
}
