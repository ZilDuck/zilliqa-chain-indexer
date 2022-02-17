package entity

import (
	"errors"
	"fmt"
	"github.com/gosimple/slug"
	"regexp"
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

	HasMetadata   bool        `json:"hasMetadata"`
	MetadataError string      `json:"metadataError"`
	Metadata      interface{} `json:"metadata"`
}

func (n Nft) Slug() string {
	return CreateNftSlug(n.TokenId, n.Contract)
}

func CreateNftSlug(tokenId uint64, contract string) string {
	return slug.Make(fmt.Sprintf("nft-%d-%s", tokenId, contract))
}

func (n Nft) MetadataUri() (string, error) {
	var metadataUri string
	if n.TokenUri != "" {
		metadataUri = n.TokenUri
	} else {
		metadataUri = fmt.Sprintf("%s%d", n.BaseUri, n.TokenId)
	}

	if ipfs := getIpfs(metadataUri); ipfs != "" {
		metadataUri = ipfs
	}

	if len(metadataUri)<4 || metadataUri[:4] != "http" {
		return "", errors.New("invalid metadata")
	}

	return metadataUri, nil
}

func getIpfs(metadataUri string) string {
	if len(metadataUri)<7 {
		return ""
	}

	if metadataUri[:7] == "ipfs://" {
		return metadataUri
	}

	re := regexp.MustCompile("(Qm[1-9A-HJ-NP-Za-km-z]{44})")
	parts := re.FindStringSubmatch(metadataUri)
	if len(parts) == 2 {
		return "ipfs://" + parts[1]
	}

	return ""
}