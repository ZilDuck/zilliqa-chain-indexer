package main

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config/di"
	"os"

	"github.com/sarulabs/dingo/v4"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("usage: go run main.go path/to/output/directory")
		os.Exit(1)
	}

	err := dingo.GenerateContainer((*di.Provider)(nil), args[0])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
