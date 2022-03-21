package main

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"os"
)

var container *dic.Container

func main() {
	config.Init()
	container, _ = dic.NewContainer()

	container.GetMetadataIndexer()
	container.GetElastic().InstallMappings()

	if len(os.Args) > 1 {
		c, err := container.GetContractRepo().GetContractByAddress(os.Args[1])
		if err != nil {
			zap.S().Fatalf("Failed to find contract: %s", os.Args[1])
			return
		}

		if len(os.Args) > 2 && os.Args[2] == "true" {
			_ = container.GetNftRepo().PurgeContract(c.Address)
		}

		importNftsForContract(*c)
	} else {
		importAllNfts()
	}

	container.GetElastic().Persist()

	zap.L().Info("Ready for exit")
	fmt.Scanln()
}

func importAllNfts() {
	size := 100
	page := 1

	for {
		contracts, total, err := container.GetContractRepo().GetAllNftContracts(size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts")
			panic(err)
		}
		if page == 1 {
			zap.S().Infof("Found %d Contracts", total)
		}
		if len(contracts) == 0 {
			break
		}
		for _, c := range contracts {
			if c.BlockNum < 1720508 {
				continue
			}
			//if c.Address == "0x3fe64e8b3e9e110db331b32ea26e191c07f14f80" || c.Address == "0x8a79bac7a6383211ae902f34e86c6b729906346d" {
			//	continue
			//}
			importNftsForContract(c)
		}
		container.GetElastic().BatchPersist()
		page++
	}
}

func importNftsForContract(contract entity.Contract) {
	zap.L().Info("*** Import Nfts For Contract: "+contract.Address)
	container.GetNftRepo().PurgeActions(contract.Address)

	if contract.MatchesStandard(entity.ZRC6) {
		zap.L().With(zap.String("contractAddr", contract.Address), zap.String("shape", "ZRC6")).Info("Import nfts for contract")
		if err := container.GetZrc6Indexer().IndexContract(contract); err != nil {
			zap.S().Fatalf("Failed to index ZRC6 NFTs for contract %s", contract.Address)
		}
	} else if contract.MatchesStandard(entity.ZRC1) {
		zap.L().With(zap.String("contractAddr", contract.Address), zap.String("shape", "ZRC1")).Info("Import nfts for contract")
		if err := container.GetZrc1Indexer().IndexContract(contract); err != nil {
			zap.S().Fatalf("Failed to index ZRC1 NFTs for contract %s", contract.Address)
		}
	}
}
