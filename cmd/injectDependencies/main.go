package main

import (
	"fmt"
	"github.com/dantudor/zil-indexer/internal/config/di"
	"os"

	"github.com/sarulabs/dingo/v4"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: go run main.go path/to/output/directory")
		os.Exit(1)
	}

	err := dingo.GenerateContainer((*di.Provider)(nil), os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
