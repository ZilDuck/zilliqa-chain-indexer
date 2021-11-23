package dev

import (
	"encoding/json"
	"log"
)

func Dump(el interface{}) {
	//if config.Get().Debug {
	elJson, _ := json.MarshalIndent(el, "", "  ")
	log.Println(string(elJson))
	//}
}
func DD(el interface{}) {
	//if config.Get().Debug {
	elJson, _ := json.MarshalIndent(el, "", "  ")
	log.Println(string(elJson))
	//}
	panic(nil)
}
