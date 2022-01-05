package factory

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type toTokenUri struct {
	ArgTypes    interface{} `json:"argtypes,omitempty"`
	Arguments   interface{} `json:"arguments,omitempty"`
	Constructor string      `json:"constructor,omitempty"`
}

func CreateZrc6FromBatchMint(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	if !c.ZRC6 {
		return nfts, nil
	}

	if tx.HasEventLog(entity.ZRC6BatchMintEvent) {
		for _, event := range tx.GetEventLogs(entity.ZRC6BatchMintEvent) {

			var toTokenUris []toTokenUri
			toTokenUriPairList, err := event.Params.GetParam("to_token_uri_pair_list")
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to get to_token_uri_pair_list")
			}

			if err := json.Unmarshal([]byte(toTokenUriPairList.Value.Primitive.(string)), &toTokenUris); err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to unmarshal to_token_uri_pair_list")
			}

			startId, err := event.Params.GetParam("start_id")
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to get start_id")
			}

			nextTokenId, err := strconv.ParseUint(startId.Value.Primitive.(string), 10, 64)
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to convert start_id to uint64")
			}

			name, _ := c.Data.Params.GetParam("name")
			symbol, _ := c.Data.Params.GetParam("symbol")
			initialBaseUri, err := c.Data.Params.GetParam("initial_base_uri")
			if err != nil {
				return nil, err
			}

			for _, i := range toTokenUris {
				arguments := i.Arguments.([]interface{})
				if len(arguments) != 2 {
					zap.L().With(zap.Error(err)).Error("Incorrectly formatted to_token_uri_pair_list")
				}
				recipient := arguments[0].(string)
				tokenUri := arguments[1].(string)
				if tokenUri == "" {
					tokenUri = initialBaseUri.Value.Primitive.(string)
				}

				nft := entity.NFT{
					Contract: c.Address,
					TxID:     tx.ID,
					BlockNum: tx.BlockNum,
					Name:     name.Value.Primitive.(string),
					Symbol:   symbol.Value.Primitive.(string),
					TokenId:  nextTokenId,
					TokenUri: strings.TrimSpace(tokenUri),
					Owner:    strings.ToLower(recipient),
					Zrc6:     true,
				}

				nfts = append(nfts, nft)
				nextTokenId++
			}
		}
	}

	return nfts, nil
}
