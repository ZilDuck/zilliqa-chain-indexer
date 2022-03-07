package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type ContractState struct {
	Address  string `json:"address"`
	State    string `json:"state"`
}

func (c ContractState) Slug() string {
	return CreateContractStateSlug(c.Address)
}

func CreateContractStateSlug(contract string) string {
	return slug.Make(fmt.Sprintf("state-%s", contract))
}
