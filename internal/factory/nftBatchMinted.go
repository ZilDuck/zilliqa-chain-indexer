package factory

import (
	"encoding/json"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"go.uber.org/zap"
	"strings"
)

func CreateNftsFromBatchMintingTx(tx entity.Transaction, c entity.Contract, nextTokenId uint64) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	if !c.ZRC6 {
		return nfts, nil
	}

	if tx.HasTransition(entity.TransitionZRC6BatchMintCallback) {
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

				recipientBech32, _ := bech32.ToBech32Address(recipient)

				name, _ := c.Data.Params.GetParam("name")
				symbol, _ := c.Data.Params.GetParam("symbol")

				nft := entity.NFT{
					Contract:        c.Address,
					ContractBech32:  c.AddressBech32,
					Name:            name.Value.Primitive.(string),
					Symbol:          symbol.Value.Primitive.(string),
					TxID:            tx.ID,
					BlockNum:        tx.BlockNum,
					TokenId:         nextTokenId,
					TokenUri:        fmt.Sprintf("%s%d", strings.TrimSpace(tokenUri.Value.Primitive.(string)), nextTokenId),
					By:              recipient,
					ByBech32:        recipientBech32,
					Recipient:       recipient,
					RecipientBech32: recipientBech32,
					Owner:           recipient,
					OwnerBech32:     recipientBech32,
				}

				zap.L().With(
					zap.String("recipient", recipient),
					zap.Uint64("blockNum", tx.BlockNum),
					zap.String("symbol", nft.Symbol),
					zap.Uint64("tokenId", nft.TokenId),
				).Info("Batch Index NFT")

				nfts = append(nfts, nft)
				nextTokenId++
			}
		}
	}

	return nfts, nil
}
