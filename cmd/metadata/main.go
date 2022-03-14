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
)

var (
	messageService messenger.MessageService
	metadataIndexer indexer.MetadataIndexer
	elastic elastic_search.Index
	nftRepo repository.NftRepository
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	metadataIndexer = container.GetMetadataIndexer()
	elastic = container.GetElastic()
	nftRepo = container.GetNftRepo()

	zap.L().Info("Subscribing to metadata refresh")

	messages := make(chan *sqs.Message, 10)

	go container.GetMessenger().PollMessages(messenger.MetadataRefresh, messages)

	for message := range messages {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(*message.Body), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
			continue
		}

		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh")

		if err := messageService.DeleteMessage(messenger.MetadataRefresh, message); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to delete message")
		}

		err := metadataIndexer.RefreshMetadata(data.Contract, data.TokenId)
		if err != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("Metadata refresh failed")
			continue
		}

		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh success")
		elastic.Persist()

		nft, err := nftRepo.GetNft(data.Contract, data.TokenId)
		if err == nil {
			metadataIndexer.TriggerAssetRefresh(*nft)
		}
	}
}
