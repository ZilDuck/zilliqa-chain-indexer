package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type ContractState struct {
	Address string         `json:"address"`
	State   []StateElement `json:"state"`
}

type StateElement struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c ContractState) Slug() string {
	return CreateStateSlug(c.Address)
}

func CreateStateSlug(contract string) string {
	return slug.Make(fmt.Sprintf("state-%s", contract))
}

func (c ContractState) GetElement(key string) (string, bool) {
	for _, el := range c.State {
		if el.Key == key {
			return el.Value, true
		}
	}

	return "", false
}
