package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type ContractState struct {
	Address  string                 `json:"address"`
	BlockNum uint64                 `json:"blockNum"`
	State    []ContractStateElement `json:"state"`
}

type ContractStateElement struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c ContractState) Slug() string {
	return CreateContractStateSlug(c.Address)
}

func CreateContractStateSlug(contract string) string {
	return slug.Make(fmt.Sprintf("state-%s", contract))
}
