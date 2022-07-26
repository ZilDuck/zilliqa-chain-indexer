package di

import (
	"crypto/tls"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/bunny"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/daemon"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/patrickmn/go-cache"
	"github.com/sarulabs/dingo/v4"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var Definitions = []dingo.Def{
	{
		Name: "elastic",
		Build: func() (elastic_search.Index, error) {
			elastic, err := elastic_search.New()
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
			rpcClient, err := zilliqa.NewClient(
				config.Get().Zilliqa.Url,
				config.Get().Zilliqa.Timeout,
				config.Get().Zilliqa.Debug,
			)
			if err != nil {
				return nil, err
			}

			return zilliqa.NewZilliqaService(zilliqa.NewProvider(rpcClient)), nil
		},
	},
	{
		Name: "sqs",
		Build: func() (*sqs.SQS, error) {
			sess := session.Must(session.NewSession(&aws.Config{
				Credentials: credentials.NewStaticCredentials(config.Get().Aws.AccessKey, config.Get().Aws.SecretKey, ""),
				Region:      aws.String(config.Get().Aws.Region),
			}))

			return sqs.New(sess), nil
		},
	},
	{
		Name: "messenger",
		Build: func(sqs *sqs.SQS) (messenger.MessageService, error) {
			return messenger.NewMessenger(sqs), nil
		},
	},
	{
		Name: "daemon",
		Build: func(
			elastic elastic_search.Index,
			indexer indexer.Indexer,
			zilliqa zilliqa.Service,
			txRepo repository.TransactionRepository,
			nftRepo repository.NftRepository,
			contractRepo repository.ContractRepository,
			contractIndexer indexer.ContractIndexer,
			zrc1Indexer indexer.Zrc1Indexer,
			zrc6Indexer indexer.Zrc6Indexer,
			marketplaceIndexer indexer.MarketplaceIndexer,
			metadataIndexer indexer.MetadataIndexer,
		) (*daemon.Daemon, error) {
			return daemon.NewDaemon(elastic, config.Get().FirstBlockNum, indexer, zilliqa, txRepo, nftRepo, contractRepo, contractIndexer, zrc1Indexer, zrc6Indexer, marketplaceIndexer, metadataIndexer), nil
		},
	},
	{
		Name: "indexer",
		Build: func(
			elastic elastic_search.Index,
			txIndexer indexer.TransactionIndexer,
			contractIndexer indexer.ContractIndexer,
			zrc1Indexer indexer.Zrc1Indexer,
			zrc6Indexer indexer.Zrc6Indexer,
			marketplaceIndexer indexer.MarketplaceIndexer,
			txRepo repository.TransactionRepository,
			cache *cache.Cache,
		) (indexer.Indexer, error) {
			return indexer.NewIndexer(config.Get().BulkIndex.Size, elastic, txIndexer, contractIndexer, zrc1Indexer, zrc6Indexer, marketplaceIndexer, txRepo, cache), nil
		},
	},
	{
		Name: "tx.indexer",
		Build: func(
			zilliqa zilliqa.Service,
			elastic elastic_search.Index,
			transactionFactory factory.TransactionFactory,
			txRepo repository.TransactionRepository,
		) (indexer.TransactionIndexer, error) {
			return indexer.NewTransactionIndexer(zilliqa, elastic, transactionFactory, txRepo), nil
		},
	},
	{
		Name: "contract.indexer",
		Build: func(
			elastic elastic_search.Index,
			zilliqa zilliqa.Service,
			factory factory.ContractFactory,
			txRepo repository.TransactionRepository,
			contractRepo repository.ContractRepository,
			nftRepo repository.NftRepository,
			metadataService metadata.Service,
		) (indexer.ContractIndexer, error) {
			return indexer.NewContractIndexer(elastic, zilliqa, factory, txRepo, contractRepo, nftRepo, metadataService), nil
		},
	},
	{
		Name: "zrc1.indexer",
		Build: func(
			elastic elastic_search.Index,
			contractRepo repository.ContractRepository,
			contractStateRepo repository.ContractStateRepository,
			nftRepo repository.NftRepository,
			txRepo repository.TransactionRepository,
			factory factory.Zrc1Factory,
		) (indexer.Zrc1Indexer, error) {
			return indexer.NewZrc1Indexer(elastic, contractRepo, contractStateRepo, nftRepo, txRepo, factory), nil
		},
	},
	{
		Name: "zrc6.indexer",
		Build: func(
			elastic elastic_search.Index,
			contractRepo repository.ContractRepository,
			nftRepo repository.NftRepository,
			txRepo repository.TransactionRepository,
			factory factory.Zrc6Factory,
		) (indexer.Zrc6Indexer, error) {
			return indexer.NewZrc6Indexer(elastic, contractRepo, nftRepo, txRepo, factory), nil
		},
	},
	{
		Name: "metadata.indexer",
		Build: func(
			elastic elastic_search.Index,
			nftRepo repository.NftRepository,
			contractRepo repository.ContractRepository,
			messageService messenger.MessageService,
			metadataService metadata.Service,
		) (indexer.MetadataIndexer, error) {
			return indexer.NewMetadataIndexer(elastic, nftRepo, contractRepo, messageService, metadataService), nil
		},
	},
	{
		Name: "marketplace.indexer",
		Build: func(
			elastic elastic_search.Index,
			nftRepo repository.NftRepository,
			contractRepo repository.ContractRepository,
			contractStateRepo repository.ContractStateRepository,
			zilkroadMarketplaceFactory factory.ZilkroadMarketplaceFactory,
			okimotoMarketplaceFactory factory.OkimotoMarketplaceFactory,
			arkyMarketplaceFactory factory.ArkyMarketplaceFactory,
			mintableMarketplaceFactory factory.MintableMarketplaceFactory,
		) (indexer.MarketplaceIndexer, error) {
			return indexer.NewMarketplaceIndexer(elastic, nftRepo, contractRepo, contractStateRepo, zilkroadMarketplaceFactory, okimotoMarketplaceFactory, arkyMarketplaceFactory, mintableMarketplaceFactory), nil
		},
	},
	{
		Name: "tx.repo",
		Build: func(elastic elastic_search.Index) (repository.TransactionRepository, error) {
			return repository.NewTransactionRepository(elastic), nil
		},
	},
	{
		Name: "contract.repo",
		Build: func(elastic elastic_search.Index) (repository.ContractRepository, error) {
			return repository.NewContractRepository(elastic), nil
		},
	},
	{
		Name: "contractMetadata.repo",
		Build: func(elastic elastic_search.Index) (repository.ContractMetadataRepository, error) {
			return repository.NewContractMetadataRepository(elastic), nil
		},
	},
	{
		Name: "contractState.repo",
		Build: func(elastic elastic_search.Index) (repository.ContractStateRepository, error) {
			return repository.NewContractStateRepository(elastic), nil
		},
	},
	{
		Name: "nft.repo",
		Build: func(elastic elastic_search.Index) (repository.NftRepository, error) {
			return repository.NewNftRepository(elastic), nil
		},
	},
	{
		Name: "nftAction.repo",
		Build: func(elastic elastic_search.Index) (repository.NftActionRepository, error) {
			return repository.NewNftActionRepository(elastic), nil
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
	{
		Name: "zrc1.factory",
		Build: func() (factory.Zrc1Factory, error) {
			return factory.NewZrc1Factory(config.Get().ContractsWithoutMetadata), nil
		},
	},
	{
		Name: "zrc6.factory",
		Build: func() (factory.Zrc6Factory, error) {
			return factory.NewZrc6Factory(config.Get().ContractsWithoutMetadata), nil
		},
	},
	{
		Name: "marketplace.zilkroad.factory",
		Build: func(nftRepo repository.NftRepository, contractRepo repository.ContractRepository, stateRepo repository.ContractStateRepository) (factory.ZilkroadMarketplaceFactory, error) {
			return factory.NewZilkroadMarketplaceFactory(nftRepo, contractRepo, stateRepo), nil
		},
	},
	{
		Name: "marketplace.okimoto.factory",
		Build: func(nftRepo repository.NftRepository, nftActionRepo repository.NftActionRepository, stateRepository repository.ContractStateRepository) (factory.OkimotoMarketplaceFactory, error) {
			return factory.NewOkimotoMarketplaceFactory(nftRepo, nftActionRepo, stateRepository), nil
		},
	},
	{
		Name: "marketplace.arky.factory",
		Build: func(nftRepo repository.NftRepository, stateRepo repository.ContractStateRepository) (factory.ArkyMarketplaceFactory, error) {
			return factory.NewArkyMarketplaceFactory(nftRepo, stateRepo), nil
		},
	},
	{
		Name: "marketplace.mintable.factory",
		Build: func(nftRepo repository.NftRepository, nftActionRepo repository.NftActionRepository) (factory.MintableMarketplaceFactory, error) {
			return factory.NewMintableMarketplaceFactory(nftRepo, nftActionRepo), nil
		},
	},
	{
		Name: "metadata.service",
		Build: func() (metadata.Service, error) {
			retryClient := retryablehttp.NewClient()
			retryClient.Logger = nil
			retryClient.RetryMax = config.Get().MetadataRetries
			retryClient.HTTPClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			return metadata.NewMetadataService(retryClient, config.Get().Ipfs.Hosts, config.Get().Ipfs.Timeout), nil
		},
	},
	{
		Name: "bunny.service",
		Build: func() (bunny.Service, error) {
			retryClient := retryablehttp.NewClient()
			retryClient.Logger = nil
			retryClient.HTTPClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			return bunny.NewService(config.Get().Bunny.CdnUrl, config.Get().Bunny.AccessKey, retryClient), nil
		},
	},
}
