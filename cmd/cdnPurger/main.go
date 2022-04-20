package main

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/bunny"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"go.uber.org/zap"
)

var (
	bunnyService bunny.Service
)

func main()  {
	config.Init("cdnPurger")

	container, _ := dic.NewContainer()
	bunnyService = container.GetBunny()

	zap.L().Info("Subscribing to cdn purge")
	if err := container.GetMessenger().ConsumeMessages(messenger.CdnPurge, purge()); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to consume messages")
	}
}

func purge() func(string) {
	return func(msg string) {
		var data messenger.Nft
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to read message")
			return
		}

		err := bunnyService.Purge(data.Contract, data.TokenId)
		if err != nil {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId), zap.Error(err)).Error("CDN Purge failed")
		} else {
			zap.L().With(zap.String("contract", data.Contract), zap.Uint64("tokenId", data.TokenId)).Info("CDN Purge success")
		}
	}
}