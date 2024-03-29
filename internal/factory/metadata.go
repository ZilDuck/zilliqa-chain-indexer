package factory

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"go.uber.org/zap"
	"time"
)

func GetMetadata(nft entity.Nft) *entity.Metadata {
	uri := GetMetadataUri(nft)

	if ipfs := helper.GetIpfs(uri, nil); ipfs != nil {
		return &entity.Metadata{Uri: *ipfs, IsIpfs: true, Status: entity.MetadataPending, CreatedAt: time.Now()}
	}

	if !helper.IsUrl(uri) {
		zap.L().With(zap.String("uri", uri), zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Warn("invalid uri")
		return &entity.Metadata{Uri: uri, Error: "invalid uri", Status: entity.MetadataFailure, CreatedAt: time.Now()}
	}

	return &entity.Metadata{Uri: uri, IsIpfs: false, Status: entity.MetadataPending, CreatedAt: time.Now()}
}

func GetMetadataUri(nft entity.Nft) string {
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