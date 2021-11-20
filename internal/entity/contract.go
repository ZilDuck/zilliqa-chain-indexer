package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type TRANSITION string

var (
	TransitionRegenerateDuck          TRANSITION = "regenerateDuck"
	TransitionRecipientAcceptTransfer TRANSITION = "RecipientAcceptTransfer"
	TransitionZRC6BatchMintCallback   TRANSITION = "ZRC6_BatchMintCallback"
	TransitionZRC6SetBaseURICallback  TRANSITION = "ZRC6_SetBaseURICallback"
)

type Contract struct {
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	AddressBech32   string   `json:"addressBech32"`
	BlockNum        uint64   `json:"blockNum"`
	Code            string   `json:"code"`
	Data            Data     `json:"data"`
	MutableParams   Params   `json:"mutableParams"`
	ImmutableParams Params   `json:"immutableParams"`
	Transitions     []string `json:"transitions"`
	ZRC1            bool     `json:"zrc1"`
	ZRC6            bool     `json:"zrc6"`
	Minters         []string `json:"minters"`
}

func (c Contract) Slug() string {
	return CreateContractSlug(c.Address)
}

func CreateContractSlug(contract string) string {
	return slug.Make(fmt.Sprintf("contract-%s", contract))
}

func (c Contract) HasTransition(t TRANSITION) bool {
	for idx, _ := range c.Transitions {
		if len(c.Transitions[idx]) > len(t) && c.Transitions[idx][:len(t)] == string(t) {
			return true
		}
	}

	return false
}
