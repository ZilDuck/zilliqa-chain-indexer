package main

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

var (
	messageService messenger.MessageService
	zrc6Indexer indexer.Zrc6Indexer
	elastic elastic_search.Index
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	messageService = container.GetMessenger()
	zrc6Indexer = container.GetZrc6Indexer()
	elastic = container.GetElastic()

	go pollMetadataRefresh()
	go pollAssetRefresh()

	for {switch {}}
}

func pollMetadataRefresh() {
	zap.L().Info("Subscribing to metadata refresh")
	messages := make(chan *sqs.Message, 10)
	go messageService.PollMessages(messenger.MetadataRefresh, messages)

	for message := range messages {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(*message.Body), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
		}
		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh")

		if err := zrc6Indexer.RefreshMetadata(data.Contract, data.TokenId); err != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("Metadata refresh failed")
		} else {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Info("Metadata refresh success")
		}
		if err := messageService.DeleteMessage(messenger.MetadataRefresh, message); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to delete message")
		}
		elastic.Persist()

		zrc6Indexer.TriggerAssetRefresh(entity.Nft{Contract: data.Contract, TokenId: data.TokenId})
	}
}

func pollAssetRefresh() {
	zap.L().Info("Subscribing to asset refresh")
	messages := make(chan *sqs.Message, 10)
	go messageService.PollMessages(messenger.AssetRefresh, messages)

	for message := range messages {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(*message.Body), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
		}
		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Asset refresh")

		if err := zrc6Indexer.RefreshAsset(data.Contract, data.TokenId); err != nil {
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