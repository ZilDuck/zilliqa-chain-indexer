package di

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/daemon"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_cache"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
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
			contractRepo repository.ContractRepository,
			contractIndexer indexer.ContractIndexer,
			zrc1Indexer indexer.Zrc1Indexer,
			zrc6Indexer indexer.Zrc6Indexer,
		) (*daemon.Daemon, error) {
			return daemon.NewDaemon(elastic, config.Get().FirstBlockNum, indexer, zilliqa, txRepo, contractRepo, contractIndexer, zrc1Indexer, zrc6Indexer), nil
		},
	},
	{
		Name: "indexer",
		Build: func(
			elastic elastic_cache.Index,
			txIndexer indexer.TransactionIndexer,
			contractIndexer indexer.ContractIndexer,
			zrc1Indexer indexer.Zrc1Indexer,
			zrc6Indexer indexer.Zrc6Indexer,
			txRepo repository.TransactionRepository,
			cache *cache.Cache,
		) (indexer.Indexer, error) {
			return indexer.NewIndexer(config.Get().BulkIndexSize, elastic, txIndexer, contractIndexer, zrc1Indexer, zrc6Indexer, txRepo, cache), nil
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
			contractRepo repository.ContractRepository,
			nftRepo repository.NftRepository,
		) (indexer.ContractIndexer, error) {
			return indexer.NewContractIndexer(elastic, factory, txRepo, contractRepo, nftRepo), nil
		},
	},
	{
		Name: "zrc1.indexer",
		Build: func(
			elastic elastic_cache.Index,
			contractRepo repository.ContractRepository,
			nftRepo repository.NftRepository,
			txRepo repository.TransactionRepository,
		) (indexer.Zrc1Indexer, error) {
			return indexer.NewZrc1Indexer(elastic, contractRepo, nftRepo, txRepo), nil
		},
	},
	{
		Name: "zrc6.indexer",
		Build: func(
			elastic elastic_cache.Index,
			contractRepo repository.ContractRepository,
			nftRepo repository.NftRepository,
			txRepo repository.TransactionRepository,
		) (indexer.Zrc6Indexer, error) {
			return indexer.NewZrc6Indexer(elastic, contractRepo, nftRepo, txRepo), nil
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
