package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/asset"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	config.Init("asset")
	container, _ := dic.NewContainer()

	router := asset.NewServer(
		container.GetNftRepo(),
		container.GetContractMetadataRepo(),
		container.GetMetadataService(),
	).Router()

	zap.L().Info("Serving assets on :" + config.Get().AssetPort)

	if err := http.ListenAndServe(":"+config.Get().AssetPort, router); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to start asset server")
	}
}
