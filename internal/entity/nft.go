package entity

import (
	"fmt"
	"github.com/gosimple/slug"
)

type Nft struct {
	Contract string `json:"contract"`
	TxID     string `json:"txId"`
	BlockNum uint64 `json:"blockNum"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	TokenId  uint64 `json:"tokenId"`
	BaseUri  string `json:"baseUri"`
	TokenUri string `json:"tokenUri"`
	MediaUri string `json:"mediaUri"`
	Owner    string `json:"owner"`
	BurnedAt uint64 `json:"burnedAt"`
	Zrc1     bool   `json:"zrc1"`
	Zrc6     bool   `json:"zrc6"`
}

func (n Nft) Slug() string {
	return CreateNftSlug(n.TokenId, n.Contract)
}

func (n Nft) MetadataUri() string {
	if n.TokenUri != "" {
		return n.TokenUri
	}
	return fmt.Sprintf("%s%d", n.BaseUri, n.TokenId)
}

func CreateNftSlug(tokenId uint64, contract string) string {
	return slug.Make(fmt.Sprintf("nft-%d-%s", tokenId, contract))
}
