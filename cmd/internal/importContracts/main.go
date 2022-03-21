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
		contractAddr := os.Args[1]
		_, err := container.GetContractRepo().GetContractByAddress(contractAddr)
		if err != nil {
			zap.S().Fatalf("Failed to find contract: %s", os.Args[1])
			return
		}

		tx, err := container.GetTxRepo().GetContractCreationForContract(contractAddr)
		if err != nil {
			zap.S().Fatalf("Failed to find contract creation tx for %s", contractAddr)
			return
		}

		container.GetContractIndexer().Index([]entity.Transaction{*tx})
		container.GetElastic().Persist()
		return
	}

	if err := container.GetContractIndexer().BulkIndex(1800000); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}
