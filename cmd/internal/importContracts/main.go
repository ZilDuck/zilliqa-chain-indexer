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
	config.Init("importContracts")
	container, _ = dic.NewContainer()

	if len(os.Args) == 2 {
		contractAddr := os.Args[1]

		tx, err := container.GetTxRepo().GetContractCreationForContract(contractAddr)
		if err != nil {
			zap.S().Fatalf("Failed to find contract creation tx for %s", contractAddr)
			return
		}

		container.GetContractIndexer().Index([]entity.Transaction{*tx})
		container.GetElastic().Persist()
		return
	}

	if err := container.GetContractIndexer().BulkIndex(config.Get().FirstBlockNum); err != nil {
		zap.L().Fatal("Failed to bulk index contracts")
	}
}
