package factory

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type Zrc6Factory interface {
	CreateFromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error)
	CreateFromBatchMint(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error)
}

type zrc6Factory struct {
	contractsWithoutMetadata map[string]string
}

func NewZrc6Factory(contractsWithoutMetadata map[string]string) Zrc6Factory {
	return zrc6Factory{contractsWithoutMetadata}
}

type toTokenUri struct {
	ArgTypes    interface{} `json:"argtypes,omitempty"`
	Arguments   interface{} `json:"arguments,omitempty"`
	Constructor string      `json:"constructor,omitempty"`
}


func (f zrc6Factory) CreateFromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error) {
	nfts := make([]entity.Nft, 0)

	for _, event := range tx.GetEventLogs(entity.ZRC6MintEvent) {
		name, _ := c.Data.Params.GetParam("name")
		symbol, _ := c.Data.Params.GetParam("symbol")

		tokenId, err := GetTokenId(event.Params)
		if err != nil {
			return nil, err
		}

		to, err := getPrimitiveParam(event.Params, "to")
		if err != nil {
			return nil, err
		}

		tokenUri, _ := getNftTokenUri(event.Params, tx)

		nft := entity.Nft{
			Contract:  c.Address,
			TxID:      tx.ID,
			BlockNum:  tx.BlockNum,
			Name:      name.Value.Primitive.(string),
			Symbol:    symbol.Value.Primitive.(string),
			TokenId:   tokenId,
			BaseUri:   c.BaseUri,
			TokenUri:  tokenUri,
			Owner:     strings.ToLower(to),
			Zrc6:      true,
		}

		if f.contractHasMetadata(c) {
			nft.HasMetadata = true
			nft.Metadata = GetMetadata(nft)
		} else {
			nft.HasMetadata = false
			nft.Metadata = nil
			if helper.IsIpfs(nft.TokenUri) {
				ipfsUri := *helper.GetIpfs(nft.TokenUri, &c)
				if val, exists := f.contractsWithoutMetadata[nft.Contract]; exists {
					nft.AssetUri = val + ipfsUri[7:]
				} else {
					nft.AssetUri = *helper.GetIpfs(nft.TokenUri, &c)
				}
			} else {
				nft.AssetUri = nft.TokenUri
			}
		}

		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func (f zrc6Factory) CreateFromBatchMint(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error) {
	nfts := make([]entity.Nft, 0)

	if !c.MatchesStandard(entity.ZRC6) {
		return nfts, nil
	}

	if tx.HasEventLog(entity.ZRC6BatchMintEvent) {
		for _, event := range tx.GetEventLogs(entity.ZRC6BatchMintEvent) {
			name, _ := c.Data.Params.GetParam("name")
			symbol, _ := c.Data.Params.GetParam("symbol")

			startId, err := event.Params.GetParam("start_id")
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to get start_id")
				continue
			}

			nextTokenId, err := strconv.ParseUint(startId.Value.Primitive.(string), 10, 64)
			if err != nil {
				zap.L().With(zap.Error(err)).Error("Failed to convert start_id to uint64")
			}

			var toTokenUris []toTokenUri
			if event.Params.HasParam("to_token_uri_pair_list") {
				toTokenUriPairList, err := event.Params.GetParam("to_token_uri_pair_list")
				if err != nil {
					zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contract", c.Address)).Error("Failed to get to_token_uri_pair_list")
					continue
				}

				if err := json.Unmarshal([]byte(toTokenUriPairList.Value.Primitive.(string)), &toTokenUris); err != nil {
					zap.L().With(zap.Error(err)).Error("Failed to unmarshal to_token_uri_pair_list")
					continue
				}

				for _, i := range toTokenUris {
					arguments := i.Arguments.([]interface{})
					if len(arguments) != 2 {
						zap.L().With(zap.Error(err)).Error("Incorrectly formatted to_token_uri_pair_list")
					}

					nft := entity.Nft{
						Contract: c.Address,
						TxID:     tx.ID,
						BlockNum: tx.BlockNum,
						Name:     name.Value.Primitive.(string),
						Symbol:   symbol.Value.Primitive.(string),
						TokenId:  nextTokenId,
						TokenUri: arguments[1].(string),
						BaseUri:  c.BaseUri,
						Owner:    strings.ToLower(arguments[0].(string)),
						Zrc6:     true,
					}

					if f.contractHasMetadata(c) {
						nft.HasMetadata = true
						nft.Metadata = GetMetadata(nft)
					} else {
						nft.HasMetadata = false
						nft.Metadata = nil
						if helper.IsIpfs(nft.TokenUri) {
							ipfsUri := *helper.GetIpfs(nft.TokenUri, &c)
							if val, exists := f.contractsWithoutMetadata[nft.Contract]; exists {
								nft.AssetUri = val + ipfsUri[7:]
							} else {
								nft.AssetUri = *helper.GetIpfs(nft.TokenUri, &c)
							}
						} else {
							nft.AssetUri = nft.TokenUri
						}
					}

					nfts = append(nfts, nft)
					nextTokenId++
				}
			}

			// If a contract uses to_list when batch minting it in NON compliant ZRC6
			var toUris []string
			if event.Params.HasParamWithType("to_list", "List (ByStr20)") {
				toList, err := event.Params.GetParam("to_list")
				if err != nil {
					zap.L().With(zap.Error(err), zap.String("txID", tx.ID), zap.String("contract", c.Address)).Error("Failed to get to_list")
					continue
				}
				if err := json.Unmarshal([]byte(toList.Value.Primitive.(string)), &toUris); err != nil {
					zap.L().With(zap.Error(err)).Error("Failed to unmarshal toList")
					continue
				}

				for _, i := range toUris {
					nft := entity.Nft{
						Contract: c.Address,
						TxID:     tx.ID,
						BlockNum: tx.BlockNum,
						Name:     name.Value.Primitive.(string),
						Symbol:   symbol.Value.Primitive.(string),
						TokenId:  nextTokenId,
						BaseUri:  c.BaseUri,
						Owner:    strings.ToLower(i),
						Zrc6:     true,
					}

					nft.Metadata = GetMetadata(nft)

					nfts = append(nfts, nft)
					nextTokenId++
				}
			}
		}
	}

	return nfts, nil
}

func (f zrc6Factory) contractHasMetadata(c entity.Contract) bool {
	_, exists := f.contractsWithoutMetadata[c.Address]
	return !exists
}

func GetTokenId(params entity.Params) (uint64, error) {
	tokenId, err := params.GetParam("token_id")
	if err != nil {
		tokenId, err = params.GetParam("token")
		if err != nil {
			return 0, err
		}
	}
	tokenIdInt, err := strconv.ParseUint(tokenId.Value.Primitive.(string), 0, 64)
	if err != nil {
		return 0, err
	}

	return tokenIdInt, nil
}

func getNftTokenUri(params entity.Params, tx entity.Transaction) (string, error) {
	if tx.HasTransition("Mint") {
		for _, ts := range tx.GetTransition("Mint") {
			if metaData, err := ts.Msg.Params.GetParam("token_metadata"); err == nil {
				return metaData.Value.Primitive.(string), nil
			}
		}
	}

	tokenUri, err := params.GetParam("token_uri")
	if err != nil {
		return "", err
	}

	return tokenUri.Value.Primitive.(string), nil
}

func getRecipient(params entity.Params) (string, error) {
	return getPrimitiveParam(params, "recipient")
}

func getPrimitiveParam(params entity.Params, name string) (string, error) {
	param, err := params.GetParam(name)
	if err != nil {
		return "", err
	}

	return param.Value.Primitive.(string), nil
}


