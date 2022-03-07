package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
)

func main() {
	config.Init()

	container, _ := dic.NewContainer()
	contractIndexer := container.GetContractIndexer()
	elastic := container.GetElastic()

	size := 10
	page := 1

	for {
		contracts, _, err := container.GetContractRepo().GetAllContracts(size, page)
		if err != nil {
			panic(err)
		}

		if len(contracts) == 0 {
			break
		}

		for _, contract := range contracts {
			if err := contractIndexer.IndexContractState(contract.Address, true); err != nil {
				panic(err)
			}
			elastic.BatchPersist()
		}
		elastic.Persist()

		page++
	}
}
