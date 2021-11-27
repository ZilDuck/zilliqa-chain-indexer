package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type TRANSITION string

const (
	ZRC1MintCallBack            TRANSITION = "MintCallBack"
	ZRC1RegenerateDuck          TRANSITION = "regenerateDuck"
	ZRC1RecipientAcceptTransfer TRANSITION = "RecipientAcceptTransfer"
	ZRC1BurnCallBack            TRANSITION = "BurnCallBack"

	ZRC6MintCallback                TRANSITION = "ZRC6_MintCallback"
	ZRC6BatchMintCallback           TRANSITION = "ZRC6_BatchMintCallback"
	ZRC6SetBaseURICallback          TRANSITION = "ZRC6_SetBaseURICallback"
	ZRC6RecipientAcceptTransferFrom TRANSITION = "ZRC6_RecipientAcceptTransferFrom"
	ZRC6BurnCallback                TRANSITION = "ZRC6_BurnCallback"
	ZRC6BatchBurnCallback           TRANSITION = "ZRC6_BatchBurnCallback"
)

var (
	Zrc1Transitions = []TRANSITION{ZRC1RegenerateDuck, ZRC1RecipientAcceptTransfer, ZRC1BurnCallBack}
	Zrc6Transitions = []TRANSITION{ZRC6MintCallback, ZRC6BatchMintCallback, ZRC6SetBaseURICallback, ZRC6RecipientAcceptTransferFrom, ZRC6BurnCallback, ZRC6BatchBurnCallback}
)

type Contract struct {
	//immutable
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

	//mutable
	BaseUri string `json:"baseuri"`
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
