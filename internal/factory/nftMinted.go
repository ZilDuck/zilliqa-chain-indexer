package factory

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

func CreateNftsFromMintingTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	if c.ZRC1 {
		return createZrc1NftsFromMintingTx(tx, c)
	}

	if c.ZRC6 {
		return createZrc6NftsFromMintingTx(tx, c)
	}

	return nil, errors.New("contract is not zrc1 or zrc6")
}

func createZrc6NftsFromMintingTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	for _, event := range tx.GetEventLogs("Mint") {
		tokenId, err := GetTokenId(event.Params)
		if err != nil {
			return nil, err
		}

		tokenUri, err := c.Data.Params.GetParam("initial_base_uri")
		if err != nil {
			return nil, err
		}

		recipient, err := getPrimitiveParam(event.Params, "to")
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
			TokenId:         tokenId,
			TokenUri:        fmt.Sprintf("%s%d", strings.TrimSpace(tokenUri.Value.Primitive.(string)), tokenId),
			By:              strings.ToLower(recipient),
			ByBech32:        recipientBech32,
			Recipient:       strings.ToLower(recipient),
			RecipientBech32: recipientBech32,
			Owner:           strings.ToLower(recipient),
			OwnerBech32:     recipientBech32,
		}

		zap.L().With(
			zap.String("recipient", recipient),
			zap.Uint64("blockNum", tx.BlockNum),
			zap.String("symbol", nft.Symbol),
			zap.Uint64("tokenId", nft.TokenId),
		).Info("Index NFT")
		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func createZrc1NftsFromMintingTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	if c.Name == "Unicutes" {
		return createNftsFromUnicuteMintingTx(tx, c)
	}

	nfts := make([]entity.NFT, 0)

	for _, mintSuccess := range tx.GetEventLogs("MintSuccess") {
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

		mintedBy, err := mintSuccess.Params.GetParam("by")
		if err != nil {
			return nil, err
		}
		mintedByBech32, _ := bech32.ToBech32Address(mintedBy.Value.Primitive.(string))
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
			TokenId:         tokenId,
			TokenUri:        tokenUri,
			By:              strings.ToLower(mintedBy.Value.Primitive.(string)),
			ByBech32:        mintedByBech32,
			Recipient:       strings.ToLower(recipient),
			RecipientBech32: recipientBech32,
			Owner:           strings.ToLower(recipient),
			OwnerBech32:     recipientBech32,
		}

		zap.L().With(
			zap.Uint64("blockNum", tx.BlockNum),
			zap.String("symbol", nft.Symbol),
			zap.Uint64("tokenId", nft.TokenId),
		).Info("Index NFT")
		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func createNftsFromUnicuteMintingTx(tx entity.Transaction, c entity.Contract) ([]entity.NFT, error) {
	nfts := make([]entity.NFT, 0)

	for _, mintSuccess := range tx.GetEventLogs("UnicuteInsertDrandValues") {
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

		mintedBy := mintSuccess.Address
		mintedByBech32, _ := bech32.ToBech32Address(mintedBy)
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
			TokenId:         tokenId,
			TokenUri:        tokenUri,
			By:              mintedBy,
			ByBech32:        mintedByBech32,
			Recipient:       recipient,
			RecipientBech32: recipientBech32,
			Owner:           recipient,
			OwnerBech32:     recipientBech32,
		}

		zap.L().With(zap.String("symbol", nft.Symbol), zap.Uint64("tokenId", nft.TokenId)).Info("Index NFT")
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
