package main

import (
	"github.com/dantudor/zil-indexer/generated/dic"
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/sarulabs/dingo/v3"
	"go.uber.org/zap"
)

var container *dic.Container

func main() {
	initialize()

	container.GetElastic().InstallMappings()

	//c, err := container.GetContractRepo().GetContactByAddress("0x06f70655d4aa5819e711563eb2383655449f24e9")
	//if err != nil {
	//	panic(err)
	//}
	//if err := container.GetNftIndexer().IndexContract(c); err != nil {
	//	panic(err)
	//}
	//container.GetElastic().Persist()
	//panic("die")

	if err := container.GetNftIndexer().BulkIndex(); err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to bulk index NFTs")
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
}
