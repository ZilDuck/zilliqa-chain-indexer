package factory

import "github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"


func CreateMintAction(nft entity.Nft) entity.NftAction {
	return entity.NftAction{
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     nft.TxID,
		BlockNum: nft.BlockNum,
		Action:   entity.MintAction,
		From:     "",
		To:       nft.Owner,
		Zrc1:     nft.Zrc1,
		Zrc6:     nft.Zrc6,
	}
}

func CreateTransferAction(nft entity.Nft, blockNum uint64, txId, buyer, seller string) entity.NftAction {
	return entity.NftAction{
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     txId,
		BlockNum: blockNum,
		Action:   entity.TransferAction,
		From:     seller,
		To:       buyer,
		Zrc1:     nft.Zrc1,
		Zrc6:     nft.Zrc6,
	}
}

func CreateMarketplaceListingAction(marketplace entity.Marketplace, nft entity.Nft, blockNum uint64, txId string, cost string, fungible string) entity.NftAction {
	return entity.NftAction{
		Marketplace: string(marketplace),
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     txId,
		BlockNum: blockNum,
		Action:   entity.MarketplaceListingAction,
		Zrc1:     nft.Zrc1,
		Zrc6:     nft.Zrc6,
		Cost:     cost,
		Fungible: fungible,
	}
}

func CreateMarketplaceDelistingAction(marketplace entity.Marketplace, nft entity.Nft, blockNum uint64, txId string) entity.NftAction {
	return entity.NftAction{
		Marketplace: string(marketplace),
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     txId,
		BlockNum: blockNum,
		Action:   entity.MarketplaceDelistingAction,
		Zrc1:     nft.Zrc1,
		Zrc6:     nft.Zrc6,
	}
}

func CreateMarketplaceSaleAction(marketplace entity.Marketplace, nft entity.Nft, blockNum uint64, txId, buyer, seller, cost, fee, royalty, royaltyBps, fungible string) entity.NftAction {
	return entity.NftAction{
		Marketplace: string(marketplace),
		Contract:    nft.Contract,
		TokenId:     nft.TokenId,
		TxID:        txId,
		BlockNum:    blockNum,
		Action:      entity.MarketplaceSaleAction,
		From:        seller,
		To:          buyer,
		Zrc1:        nft.Zrc1,
		Zrc6:        nft.Zrc6,
		Cost:        cost,
		Fee:         fee,
		Royalty:     royalty,
		RoyaltyBps:  royaltyBps,
		Fungible:    fungible,
	}
}

func CreateBurnAction(nft entity.Nft, tx entity.Transaction) entity.NftAction {
	return entity.NftAction{
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     tx.ID,
		BlockNum: tx.BlockNum,
		Action:   entity.BurnAction,
		From:     nft.Owner,
		To:       "",
		Zrc1:     nft.Zrc1,
		Zrc6:     nft.Zrc6,
	}
}
