package main

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
	"sync"
)

var (
	messageService messenger.MessageService
	metadataIndexer indexer.MetadataIndexer
	elastic elastic_search.Index
	nftRepo repository.NftRepository
)

var wg sync.WaitGroup

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	messageService = container.GetMessenger()
	metadataIndexer = container.GetMetadataIndexer()
	elastic = container.GetElastic()
	nftRepo = container.GetNftRepo()

	wg.Add(1)
	go pollMetadataRefresh()
	go pollAssetRefresh()

	wg.Wait()
}

func pollMetadataRefresh() {
	defer wg.Done()
	zap.L().Info("Subscribing to metadata refresh")
	messages := make(chan *sqs.Message, 10)
	go messageService.PollMessages(messenger.MetadataRefresh, messages)

	for message := range messages {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(*message.Body), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
		}
		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh")

		refreshDataErr := metadataIndexer.RefreshMetadata(data.Contract, data.TokenId)
		if refreshDataErr != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(refreshDataErr)).Error("Metadata refresh failed")
		} else {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh success")
		}
		if err := messageService.DeleteMessage(messenger.MetadataRefresh, message); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to delete message")
		}
		elastic.Persist()

		if refreshDataErr == nil {
			nft, err := nftRepo.GetNft(data.Contract, data.TokenId)
			if err == nil {
				metadataIndexer.TriggerAssetRefresh(*nft)
			}
		}
	}
}

func pollAssetRefresh() {
	defer wg.Done()
	zap.L().Info("Subscribing to asset refresh")
	messages := make(chan *sqs.Message, 10)
	go messageService.PollMessages(messenger.AssetRefresh, messages)

	for message := range messages {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(*message.Body), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
		}
		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Asset refresh")

		if err := metadataIndexer.RefreshAsset(data.Contract, data.TokenId); err != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("Asset refresh failed")
		} else {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Info("Asset refresh success")
		}
		if err := messageService.DeleteMessage(messenger.AssetRefresh, message); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to delete message")
		}

		elastic.Persist()
	}
}