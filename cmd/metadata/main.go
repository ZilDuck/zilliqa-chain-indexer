package main

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"go.uber.org/zap"
)

var (
	messageService messenger.MessageService
	metadataIndexer indexer.MetadataIndexer
	elastic elastic_search.Index
)

func main() {
	config.Init("metadata")

	container, _ := dic.NewContainer()
	messageService = container.GetMessenger()
	metadataIndexer = container.GetMetadataIndexer()
	elastic = container.GetElastic()

	zap.L().Info("Subscribing to metadata refresh")
	if err := messageService.ConsumeMessages(messenger.MetadataRefresh, refreshMetadata()); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to consume messages")
	}
}

func refreshMetadata() func(string) {
	return func(msg string) {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
			return
		}

		_, err := metadataIndexer.RefreshMetadata(data.Contract, data.TokenId)
		if err != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("Metadata refresh failed")
		} else {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh success")
		}
		elastic.Persist()
	}
}
