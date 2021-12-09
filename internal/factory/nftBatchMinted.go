package factory

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

func CreateZrc6FromBatchMint(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	if !c.ZRC6 {
		return nfts, nil
	}

	if tx.HasEventLog(entity.ZRC6BatchMintEvent) {
		for _, event := range tx.GetEventLogs(entity.ZRC6BatchMintEvent) {
			toListParam, err := event.Params.GetParam("to_list")
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to get toList")
			}

			var toList []string
			if err := json.Unmarshal([]byte(toListParam.Value.Primitive.(string)), &toList); err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to unmarshall toList")
			}

			startId, err := event.Params.GetParam("start_id")
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to get start_id")
			}

			nextTokenId, err := strconv.ParseUint(startId.Value.Primitive.(string), 10, 64)
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to convert start_id to uint64")
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
					Zrc6:     true,
				}

				nfts = append(nfts, nft)
				nextTokenId++
			}
		}
	}

	return nfts, nil
}
