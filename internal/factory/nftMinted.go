package factory

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"strconv"
	"strings"
)

func CreateZrc6FromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	for _, transition := range tx.GetTransition(entity.ZRC6MintCallback) {
		name, _ := c.Data.Params.GetParam("name")
		symbol, _ := c.Data.Params.GetParam("symbol")

		tokenId, err := GetTokenId(transition.Msg.Params)
		if err != nil {
			return nil, err
		}

		to, err := getPrimitiveParam(transition.Msg.Params, "to")
		if err != nil {
			return nil, err
		}

		nft := entity.NFT{
			Contract: c.Address,
			TxID:     tx.ID,
			BlockNum: tx.BlockNum,
			Name:     name.Value.Primitive.(string),
			Symbol:   symbol.Value.Primitive.(string),
			TokenId:  tokenId,
			TokenUri: c.BaseUri,
			Owner:    strings.ToLower(to),
			Zrc6:     true,
		}
		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func CreateZrc1FromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	if c.Name == "Unicutes" {
		return createUincuteFromMintTx(tx, c)
	}

	nfts := make([]entity.NFT, 0)

	for _, mintSuccess := range tx.GetEventLogs("MintSuccess") {
		name, _ := c.Data.Params.GetParam("name")
		symbol, _ := c.Data.Params.GetParam("symbol")

		tokenId, err := GetTokenId(mintSuccess.Params)
		if err != nil {
			return nil, err
		}

		tokenUri, err := getTokenUri(mintSuccess.Params, tx)
		if err != nil {
			return nil, err
		}

		recipient, err := getRecipient(mintSuccess.Params)
		if err != nil {
			return nil, err
		}

		nft := entity.NFT{
			Contract: c.Address,
			TxID:     tx.ID,
			BlockNum: tx.BlockNum,
			Name:     name.Value.Primitive.(string),
			Symbol:   symbol.Value.Primitive.(string),
			TokenId:  tokenId,
			TokenUri: tokenUri,
			Owner:    strings.ToLower(recipient),
			Zrc1:     true,
		}

		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func createUincuteFromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	for _, mintSuccess := range tx.GetEventLogs("UnicuteInsertDrandValues") {
		name, _ := c.Data.Params.GetParam("name")
		symbol, _ := c.Data.Params.GetParam("symbol")

		tokenId, err := GetTokenId(mintSuccess.Params)
		if err != nil {
			return nil, err
		}

		tokenUri, err := getTokenUri(tx.Data.Params, tx)
		if err != nil {
			return nil, err
		}

		recipient, err := getPrimitiveParam(mintSuccess.Params, "token_owner")
		if err != nil {
			return nil, err
		}

		nft := entity.NFT{
			Contract: c.Address,
			TxID:     tx.ID,
			BlockNum: tx.BlockNum,
			Name:     name.Value.Primitive.(string),
			Symbol:   symbol.Value.Primitive.(string),
			TokenId:  tokenId,
			TokenUri: tokenUri,
			Owner:    recipient,
			Zrc1:     true,
		}

		nfts = append(nfts, nft)
	}

	return nfts, nil
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

func getTokenUri(params entity.Params, tx entity.Transaction) (string, error) {
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