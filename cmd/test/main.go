package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/dev"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()

	nftRepo := container.GetNftRepo()
	metadataService := container.GetMetadataService()

	nft, _ := nftRepo.GetNft("0xd72b958b5511800ccb2ac42a512e3bfc413b36d7", 908)
	md := factory.GetMetadata(*nft)
	dev.Dump(md)

	_, err := metadataService.FetchMetadata(*nft)
	if err != nil {
		dev.DD(err.Error())
	}
}
