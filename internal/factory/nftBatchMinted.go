package factory

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"strings"
)

func CreateZrc6FromBatchMint(tx entity.Transaction, c entity.Contract, nextTokenId uint64) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	if !c.ZRC6 {
		return nfts, nil
	}

	if tx.HasTransition(entity.ZRC6BatchMintCallback) {
		if tx.Data.Tag == "BatchMint" {
			toListParam, err := tx.Data.Params.GetParam("to_list")
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to get toList")
			}
			var toList []string
			if err := json.Unmarshal([]byte(toListParam.Value.Primitive.(string)), &toList); err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to unmarshall toList")
			}

			for _, recipient := range toList {
				tokenUri, err := c.Data.Params.GetParam("initial_base_uri")
				if err != nil {
					return nil, err
				}

				name, _ := c.Data.Params.GetParam("name")
				symbol, _ := c.Data.Params.GetParam("symbol")

				nft := entity.NFT{
					Contract: c.Address,
					TxID:     tx.ID,
					BlockNum: tx.BlockNum,
					Name:     name.Value.Primitive.(string),
					Symbol:   symbol.Value.Primitive.(string),
					TokenId:  nextTokenId,
					TokenUri: strings.TrimSpace(tokenUri.Value.Primitive.(string)),
					Owner:    strings.ToLower(recipient),
				}

				nfts = append(nfts, nft)
				nextTokenId++
			}
		}
	}

	return nfts, nil
}
