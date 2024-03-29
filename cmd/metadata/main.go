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
	messageService  messenger.MessageService
	metadataIndexer indexer.MetadataIndexer
	contractIndexer indexer.ContractIndexer
	contractRepo    repository.ContractRepository
	elastic         elastic_search.Index
)

func main() {
	config.Init("metadata")

	container, _ := dic.NewContainer()
	messageService = container.GetMessenger()
	metadataIndexer = container.GetMetadataIndexer()
	contractIndexer = container.GetContractIndexer()
	contractRepo = container.GetContractRepo()
	elastic = container.GetElastic()

	messages := make(chan *sqs.Message, 10)

	go func() {
		for {
			msg := <-messages
			refreshMetadata(msg)
		}
	}()

	zap.L().Info("Subscribing to metadata refresh")
	messageService.PollMessages(messenger.MetadataRefresh, messages)
}

func refreshMetadata(msg *sqs.Message) {
	defer messageService.DeleteMessage(messenger.MetadataRefresh, msg)

	var data messenger.Nft
	if err := json.Unmarshal([]byte(*msg.Body), &data); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to read message")
		return
	}

	if data.TokenId == 0 {
		contract, err := contractRepo.GetContractByAddress(data.Contract)
		if err != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("Contract Metadata refresh failed")
		} else {
			contractIndexer.IndexContractMetadata(contract)
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Info("Contract Metadata refresh success")
			elastic.Persist()
		}
		return
	}

	_, err := metadataIndexer.RefreshMetadata(data.Contract, data.TokenId)
	if err != nil {
		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("Metadata refresh failed")
	} else {
		zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("Metadata refresh success")
		elastic.Persist()
	}

	return
}
