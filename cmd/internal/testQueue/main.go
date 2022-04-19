package main

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"go.uber.org/zap"
)

var container *dic.Container

func main() {
	config.Init("missingTx")

	container, _ = dic.NewContainer()

	_, err := container.GetMessenger().GetQueue(messenger.MetadataRefresh)
	if err != nil {
		zap.L().Error(err.Error())
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: "Contract", TokenId: 100})
	if err := container.GetMessenger().SendMessage(messenger.MetadataRefresh, msgJson, false); err != nil {
		zap.L().Error(err.Error())
	}

	container.GetMessenger().ConsumeMessages(messenger.MetadataRefresh, react())
}

func react() func(string) {
	return func(msg string) {
		var nft messenger.Nft
		if err := json.Unmarshal([]byte(msg), &nft); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get valid data")
		}
		if _, err := container.GetMetadataIndexer().RefreshMetadata(nft.Contract, nft.TokenId); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to refresh metadata")
		}
		container.GetElastic().Persist()
		return
	}
}