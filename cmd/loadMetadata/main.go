package main

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	messageService := container.GetMessenger()
	zrc6Indexer := container.GetZrc6Indexer()
	elastic := container.GetElastic()

	chnMessages := make(chan *sqs.Message, 10)
	go messageService.PollMessages(messenger.MetadataRefresh, chnMessages)

	for message := range chnMessages {
		var data messenger.RefreshMetadata
		if err := json.Unmarshal([]byte(*message.Body), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
		}
		if err := zrc6Indexer.RefreshMetadata(data.Contract, data.TokenId); err == nil {
			if err := messageService.DeleteMessage(messenger.MetadataRefresh, message); err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to delete message")
			}
		}
		elastic.Persist()
	}
}