package dev

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"log"
)

func Dump(el interface{}) {
	if config.Get().Debug || config.Get().Env == "local" {
		elJson, _ := json.MarshalIndent(el, "", "  ")
		log.Println(string(elJson))
	}
}

func DD(el interface{}) {
	if config.Get().Debug || config.Get().Env == "local" {
		elJson, _ := json.MarshalIndent(el, "", "  ")
		log.Println(string(elJson))
	}
	panic(nil)
}
