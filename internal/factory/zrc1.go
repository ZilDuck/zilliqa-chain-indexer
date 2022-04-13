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
	nfts := make([]entity.Nft, 0)
	for _, event := range tx.GetEventLogs(f.getMintEvent(c)) {
		if nft, err := f.createNftFromZrc1MintEvent(event, tx, c); err == nil {
			nfts = append(nfts, *nft)
		}
	}
	return nfts, nil
}

func (f zrc1Factory) createNftFromZrc1MintEvent(event entity.EventLog, tx entity.Transaction, c entity.Contract) (*entity.Nft, error) {
	name, _ := c.Data.Params.GetParam("name")
	symbol, _ := c.Data.Params.GetParam("symbol")

	tokenId, err := GetTokenId(event.Params)
	if err != nil {
		zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenId when minting zrc1")
		return nil, err
	}

	tokenUri, err := f.getTokenUri(event, tx, c)
	if err != nil {
		zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get tokenUri when minting zrc1")
		return nil, err
	}

	recipient, err := f.getRecipient(event, c)
	if err != nil {
		zap.L().With(zap.String("txID", tx.ID)).Warn("Failed to get recipient when minting zrc1")
		return nil, err
	}

	nft := &entity.Nft{
		Contract: c.Address,
		TxID:     tx.ID,
		BlockNum: tx.BlockNum,
		Name:     name.Value.String(),
		Symbol:   symbol.Value.String(),
		TokenId:  tokenId,
		TokenUri: tokenUri,
		Owner:    strings.ToLower(recipient),
		Zrc1:     true,
	}

	f.getMetadata(nft)

	if !nft.HasMetadata {
		// nft does not have metadata therefore the tokenUri is expected to be an asset
		// Therefore, we can populate the assetUri
		f.getAssetUri(nft, c)
	}

	return nft, nil
}

func (f zrc1Factory) getMintEvent(c entity.Contract) entity.Event {
	if c.Name == "Unicutes" {
		return entity.ZRC1UnicutesMintEvent
	}
	return entity.ZRC1MintEvent
}

func (f zrc1Factory) getRecipient(event entity.EventLog, c entity.Contract) (string, error) {
	if c.Name == "Unicutes" {
		return getPrimitiveParam(event.Params, "token_owner")
	}
	return getRecipient(event.Params)
}

func (f zrc1Factory) getTokenUri(event entity.EventLog, tx entity.Transaction, c entity.Contract) (string, error) {
	if c.Name == "Unicutes" {
		return getNftTokenUri(tx.Data.Params, tx)
	}

	return getNftTokenUri(event.Params, tx)
}

func (f zrc1Factory) getMetadata(nft *entity.Nft) {
	_, exists := f.contractsWithoutMetadata[nft.Contract]
	nft.HasMetadata = !exists
	if nft.HasMetadata {
		nft.Metadata = GetMetadata(*nft)
	}
}

func (f zrc1Factory) getAssetUri(nft *entity.Nft, c entity.Contract) {
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

