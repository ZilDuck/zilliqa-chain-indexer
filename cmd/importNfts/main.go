package main

import (
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

	container.GetElastic().InstallMappings()

	if len(os.Args) == 2 {
		c, err := container.GetContractRepo().GetContractByAddress(os.Args[1])
		if err != nil {
			zap.S().Fatalf("Failed to find contract: %s", os.Args[1])
			return
		}
		importNftsForContract(*c)
	} else {
		//importAllNfts()
	}

	container.GetElastic().Persist()
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
			zap.S().Infof("Found %d NFTs", total)
		}
		if len(contracts) == 0 {
			break
		}
		for _, c := range contracts {
			if err := container.GetZrc1Indexer().IndexContract(c); err != nil {
				zap.S().Errorf("Failed to index ZRC1 NFTs for contract %s", c.Address)
			}
			if err := container.GetZrc6Indexer().IndexContract(c); err != nil {
				zap.S().Errorf("Failed to index ZRC6 NFTs for contract %s", c.Address)
			}
		}
		container.GetElastic().BatchPersist()
		page++
	}
}

func importNftsForContract(contract entity.Contract) {
	if contract.ZRC1 {
		if err := container.GetZrc1Indexer().IndexContract(contract); err != nil {
			zap.S().Errorf("Failed to index ZRC1 NFTs for contract %s", contract.Address)
		}
	}
	if contract.ZRC6 {
		if err := container.GetZrc6Indexer().IndexContract(contract); err != nil {
			zap.S().Errorf("Failed to index ZRC6 NFTs for contract %s", contract.Address)
		}
	}
}
