package factory

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type Zrc6Factory interface {
	CreateFromMintTx(tx entity.Transaction, c entity.Contract, fetchImage bool) ([]entity.Nft, error)
	CreateFromBatchMint(tx entity.Transaction, c entity.Contract, fetchImages bool) ([]entity.Nft, error)
	FetchImage(nft *entity.Nft)
}

type zrc6Factory struct {
	metadata metadata.Service
}

func NewZrc6Factory(metadata metadata.Service) Zrc6Factory {
	return zrc6Factory{metadata}
}

type toTokenUri struct {
	ArgTypes    interface{} `json:"argtypes,omitempty"`
	Arguments   interface{} `json:"arguments,omitempty"`
	Constructor string      `json:"constructor,omitempty"`
}

func (f zrc6Factory) CreateFromMintTx(tx entity.Transaction, c entity.Contract, fetchImage bool) ([]entity.Nft, error) {
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

		tokenUri, err := getNftTokenUri(event.Params, tx)
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenUri when minting zrc1")
			continue
		}
		if tokenUri == "" {
			tokenUri = c.BaseUri
		}

		nft := entity.Nft{
			Contract: c.Address,
			TxID:     tx.ID,
			BlockNum: tx.BlockNum,
			Name:     name.Value.Primitive.(string),
			Symbol:   symbol.Value.Primitive.(string),
			TokenId:  tokenId,
			BaseUri:  c.BaseUri,
			TokenUri: tokenUri,
			Owner:    strings.ToLower(to),
			Zrc6:     true,
		}
		if fetchImage {
			f.FetchImage(&nft)
		}

		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func (f zrc6Factory) CreateFromBatchMint(tx entity.Transaction, c entity.Contract, fetchImages bool) ([]entity.Nft, error) {
	nfts := make([]entity.Nft, 0)

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

				nft := entity.Nft{
					Contract: c.Address,
					TxID:     tx.ID,
					BlockNum: tx.BlockNum,
					Name:     name.Value.Primitive.(string),
					Symbol:   symbol.Value.Primitive.(string),
					TokenId:  nextTokenId,
					BaseUri:  strings.TrimSpace(c.BaseUri),
					TokenUri: strings.TrimSpace(tokenUri),
					Owner:    strings.ToLower(recipient),
					Zrc6:     true,
				}
				if fetchImages {
					f.FetchImage(&nft)
				}

				nfts = append(nfts, nft)
				nextTokenId++
			}
		}
	}

	return nfts, nil
}

func (f zrc6Factory) FetchImage(nft *entity.Nft) {
	md, err := f.metadata.GetZrc6Metadata(*nft)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("contractAddr", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Warn("Failed to get zrc6 metadata")
	}

	if mediaUri, ok := md["image"]; ok {
		nft.MediaUri = mediaUri.(string)
	}
}

func GetTokenId(params entity.Params) (uint64, error) {
	tokenId, err := params.GetParam("token_id")
	if err != nil {
		return 0, err
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


