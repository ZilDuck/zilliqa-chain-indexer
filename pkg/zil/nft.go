package zil

import (
	"fmt"
	"github.com/gosimple/slug"
)

type NFT struct {
	Contract       string `json:"contract"`
	ContractBech32 string `json:"contractBech32"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	TxID           string `json:"txId"`
	BlockNum       uint64 `json:"blockNum"`

	TokenId  uint64 `json:"tokenId"`
	TokenUri string `json:"tokenUri"`

	By              string `json:"by"`
	ByBech32        string `json:"byBech32"`
	Recipient       string `json:"recipient"`
	RecipientBech32 string `json:"recipientBech32"`
	Owner           string `json:"owner"`
	OwnerBech32     string `json:"ownerBech32"`
}

func (n NFT) Slug() string {
	return CreateNftSlug(n.TokenId, n.Contract)
}

func CreateNftSlug(tokenId uint64, contract string) string {
	return slug.Make(fmt.Sprintf("nft-%d-%s", tokenId, contract))
}
