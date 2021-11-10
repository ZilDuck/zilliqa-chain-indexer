package nft

import (
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"github.com/dantudor/zil-indexer/pkg/zil"
	"go.uber.org/zap"
	"strconv"
)

func CreateNftsFromMintingTx(tx zil.Transaction, c zil.Contract) ([]zil.NFT, error) {
	nfts := make([]zil.NFT, 0)

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

		nft := zil.NFT{
			Contract:        c.Address,
			ContractBech32:  c.AddressBech32,
			Name:            name.Value.Primitive.(string),
			Symbol:          symbol.Value.Primitive.(string),
			TxID:            tx.ID,
			BlockNum:        tx.BlockNum,
			TokenId:         tokenId,
			TokenUri:        tokenUri,
			By:              mintedBy.Value.Primitive.(string),
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

func GetTokenId(params zil.Params) (uint64, error) {
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

func getTokenUri(params zil.Params, tx zil.Transaction) (string, error) {
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

func getRecipient(params zil.Params) (string, error) {
	recipient, err := params.GetParam("recipient")
	if err != nil {
		return "", err
	}

	return recipient.Value.Primitive.(string), nil
}
