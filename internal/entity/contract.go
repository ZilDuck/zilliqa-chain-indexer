package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type Event string

const (
	ZRC1MintEvent         Event = "MintSuccess"
	ZRC1TransferEvent     Event = "TransferSuccess"
	ZRC1TransferFromEvent Event = "TransferFromSuccess"
	ZRC1BurnEvent         Event = "BurnSuccessEvent"

	ZRC6MintEvent         Event = "Mint"
	ZRC6BatchMintEvent    Event = "BatchMint"
	ZRC6SetBaseURIEvent   Event = "SetBaseURI"
	ZRC6TransferFromEvent Event = "TransferFrom"
	ZRC6BurnEvent         Event = "Burn"
)

type Callback string

const (
	ZRC1MintCallBack            Callback = "MintCallBack"
	ZRC1RecipientAcceptTransfer Callback = "RecipientAcceptTransfer"
	ZRC1BurnCallBack            Callback = "BurnCallBack"

	ZRC6MintCallback                Callback = "ZRC6_MintCallback"
	ZRC6BatchMintCallback           Callback = "ZRC6_BatchMintCallback"
	ZRC6SetBaseURICallback          Callback = "ZRC6_SetBaseURICallback"
	ZRC6RecipientAcceptTransferFrom Callback = "ZRC6_RecipientAcceptTransferFrom"
	ZRC6BurnCallback                Callback = "ZRC6_BurnCallback"
	ZRC6BatchBurnCallback           Callback = "ZRC6_BatchBurnCallback"
)

var (
	Zrc1Callbacks = []Callback{ZRC1MintCallBack, ZRC1RecipientAcceptTransfer, ZRC1BurnCallBack}
	Zrc6Callbacks = []Callback{ZRC6MintCallback, ZRC6BatchMintCallback, ZRC6SetBaseURICallback, ZRC6RecipientAcceptTransferFrom, ZRC6BurnCallback, ZRC6BatchBurnCallback}
)

type Contract struct {
	//immutable
	Name            string               `json:"name"`
	Address         string               `json:"address"`
	BlockNum        uint64               `json:"blockNum"`
	Code            string               `json:"code"`
	Data            Data                 `json:"data"`
	MutableParams   Params               `json:"mutableParams"`
	ImmutableParams Params               `json:"immutableParams"`
	Transitions     []ContractTransition `json:"transitions"`
	ZRC1            bool                 `json:"zrc1"`
	ZRC2            bool                 `json:"zrc2"`
	ZRC6            bool                 `json:"zrc6"`

	//mutable
	BaseUri string `json:"baseuri"`
}

func (c Contract) Slug() string {
	return CreateContractSlug(c.Address)
}

func CreateContractSlug(contract string) string {
	return slug.Make(fmt.Sprintf("contract-%s", contract))
}

type ContractTransition struct {
	Index     int               `json:"index"`
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
}