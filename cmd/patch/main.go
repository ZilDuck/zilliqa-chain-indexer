package main

import (
	"encoding/json"
	"github.com/dantudor/zil-indexer/generated/dic"
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/sarulabs/dingo/v3"
	"go.uber.org/zap"
	"log"
)

var container *dic.Container

func main() {
	initialize()

	tx, _ := container.GetZilliqa().GetTransaction("903382f7935707f5edec01b8300da2444186f33d6aec91795a8e86dbba417d92")
	txJson, _ := json.Marshal(tx)
	log.Println(string(txJson))
	panic("die")
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
	zap.L().Info("Patched Started")
}
