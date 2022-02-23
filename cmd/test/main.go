package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()

	tx, err := container.GetTxRepo().GetTx("4da896d0c16aae7333cba898886c679057156fc9b5192a31adfbf5fc1511a0b0")
	if err != nil {
		panic(err)
	}

	txs := make([]entity.Transaction, 1)
	txs[0] = *tx

	nftIndexer := container.GetZrc1Indexer()
	nftIndexer.IndexTxs(txs)
	container.GetElastic().Persist()
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
