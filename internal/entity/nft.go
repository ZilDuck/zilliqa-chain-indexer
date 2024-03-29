package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type Nft struct {
	Contract    string    `json:"contract"`
	TxID        string    `json:"txId"`
	BlockNum    uint64    `json:"blockNum"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	TokenId     uint64    `json:"tokenId"`
	BaseUri     string    `json:"baseUri"`
	TokenUri    string    `json:"tokenUri"`
	Owner       string    `json:"owner"`
	BurnedAt    uint64    `json:"burnedAt"`
	Zrc1        bool      `json:"zrc1"`
	Zrc6        bool      `json:"zrc6"`
	HasMetadata bool      `json:"hasMetadata"`
	Metadata    *Metadata `json:"metadata"`
	AssetUri    string    `json:"assetUri"`

	IsDelegated    bool   `json:"isDelegated"`
	DelegatedOwner string `json:"delegatedOwner"`
}

func (n Nft) Slug() string {
	return CreateNftSlug(n.TokenId, n.Contract)
}

func CreateNftSlug(tokenId uint64, contract string) string {
	return slug.Make(fmt.Sprintf("nft-%d-%s", tokenId, contract))
}
