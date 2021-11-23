package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type NFT struct {
	Contract string `json:"contract"`
	TxID     string `json:"txId"`
	BlockNum uint64 `json:"blockNum"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	TokenId  uint64 `json:"tokenId"`
	TokenUri string `json:"tokenUri"`
	Owner    string `json:"owner"`
	BurnedAt uint64 `json:"burnedAt"`
}

func (n NFT) Slug() string {
	return CreateNftSlug(n.TokenId, n.Contract)
}

func CreateNftSlug(tokenId uint64, contract string) string {
	return slug.Make(fmt.Sprintf("nft-%d-%s", tokenId, contract))
}
