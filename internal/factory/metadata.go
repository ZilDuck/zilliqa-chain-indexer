package factory

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"go.uber.org/zap"
)

func GetMetadata(nft entity.Nft) entity.Metadata {
	uri := getMetadataUri(nft)

	if ipfs := helper.GetIpfs(uri); ipfs != nil {
		return entity.Metadata{Uri: *ipfs, Ipfs: true}
	}

	if !helper.IsUrl(uri) {
		zap.L().With(zap.String("uri", uri), zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Warn("invalid metadata uri")
		return entity.Metadata{Error: "invalid metadata uri"}
	}

	return entity.Metadata{Uri: uri, Ipfs: false}
}

func getMetadataUri(nft entity.Nft) string {
	var uri string
	if nft.Zrc6 {
		if nft.TokenUri != "" {
			uri = nft.TokenUri
		} else {
			uri = fmt.Sprintf("%s%d", nft.BaseUri, nft.TokenId)
		}
	} else if nft.Zrc1 {
		uri = nft.TokenUri
	}

	return uri
}