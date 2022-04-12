package factory

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"go.uber.org/zap"
	"strings"
)

type Zrc1Factory interface {
	CreateFromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error)
}

type zrc1Factory struct {
	contractsWithoutMetadata map[string]string
}

func NewZrc1Factory(contractsWithoutMetadata map[string]string) Zrc1Factory {
	return zrc1Factory{contractsWithoutMetadata}
}

func (f zrc1Factory) CreateFromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error) {
	if c.Name == "Unicutes" {
		return f.createUnicuteFromMintTx(tx, c)
	}

	nfts := make([]entity.Nft, 0)

	for _, event := range tx.GetEventLogs(entity.ZRC1MintEvent) {
		name, _ := c.Data.Params.GetParam("name")
		symbol, _ := c.Data.Params.GetParam("symbol")

		tokenId, err := GetTokenId(event.Params)
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenId when minting zrc1")
			continue
		}

		tokenUri, err := getNftTokenUri(event.Params, tx)
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenUri when minting zrc1")
			continue
		}

		recipient, err := getRecipient(event.Params)
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get recipient when minting zrc1")
			continue
		}

		nft := entity.Nft{
			Contract:  c.Address,
			TxID:      tx.ID,
			BlockNum:  tx.BlockNum,
			Name:      name.Value.Primitive.(string),
			Symbol:    symbol.Value.Primitive.(string),
			TokenId:   tokenId,
			TokenUri:  tokenUri,
			Owner:     strings.ToLower(recipient),
			Zrc1:      true,
		}

		if f.contractHasMetadata(c) {
			nft.Metadata = GetMetadata(nft)
		} else {
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

func (f zrc1Factory) createUnicuteFromMintTx(tx entity.Transaction, c entity.Contract) ([]entity.Nft, error) {
	nfts := make([]entity.Nft, 0)

	for _, mintSuccess := range tx.GetEventLogs("UnicuteInsertDrandValues") {
		name, _ := c.Data.Params.GetParam("name")
		symbol, _ := c.Data.Params.GetParam("symbol")

		tokenId, err := GetTokenId(mintSuccess.Params)
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenId when minting unicute")
			continue
		}

		tokenUri, err := getNftTokenUri(tx.Data.Params, tx)
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenUri when minting unicute")
			continue
		}

		recipient, err := getPrimitiveParam(mintSuccess.Params, "token_owner")
		if err != nil {
			zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get token_owner when minting unicute")
			continue
		}

		nft := entity.Nft{
			Contract:  c.Address,
			TxID:      tx.ID,
			BlockNum:  tx.BlockNum,
			Name:      name.Value.Primitive.(string),
			Symbol:    symbol.Value.Primitive.(string),
			TokenId:   tokenId,
			TokenUri:  tokenUri,
			Owner:     recipient,
			Zrc1:      true,
		}

		nft.Metadata = GetMetadata(nft)

		nfts = append(nfts, nft)
	}

	return nfts, nil
}

func (f zrc1Factory) contractHasMetadata(c entity.Contract) bool {
	_, exists := f.contractsWithoutMetadata[c.Address]
	return !exists
}
