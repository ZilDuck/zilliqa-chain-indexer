package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/dev"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()

	tx, _ := container.GetTxRepo().GetTx("4826bbeb1cfb8dcfb758e98ba2464d2e322be6ce65ef3990865e543638baa762")
	dev.DD(tx)
	//c, _ := container.GetContractFactory().CreateContractFromTx(*tx)
	//
	//dev.DD(factory.IsZrc1(*c))
	panic(nil)

	//nftRepo := container.GetNftRepo()
	//metadataService := container.GetMetadataService()
	//
	//nft, _ := nftRepo.GetNft("0xd72b958b5511800ccb2ac42a512e3bfc413b36d7", 908)
	//md := factory.GetMetadata(*nft)
	//dev.Dump(md)
	//
	//_, err := metadataService.FetchMetadata(*nft)
	//if err != nil {
	//	dev.DD(err.Error())
	//}
}
