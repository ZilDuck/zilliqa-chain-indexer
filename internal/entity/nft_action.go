package entity

import (
	"crypto/md5"
	"fmt"
)

type NftAction struct {
	Contract string `json:"contract"`
	TokenId  uint64 `json:"tokenId"`
	TxID     string `json:"txId"`
	BlockNum uint64 `json:"blockNum"`
	Action   string `json:"action"`
	From     string `json:"from"`
	To       string `json:"to"`
	Zrc1     bool   `json:"zrc1"`
	Zrc6     bool   `json:"zrc6"`
}

func (n NftAction) Slug() string {
	return CreateNftActionSlug(n.TokenId, n.Contract, n.TxID, n.Action)
}

func CreateNftActionSlug(tokenId uint64, contract, txId, action string) string {
	data := []byte(fmt.Sprintf("nftaction-%d-%s-%s-%s", tokenId, contract, txId, action))
	return fmt.Sprintf("%x", md5.Sum(data))
}