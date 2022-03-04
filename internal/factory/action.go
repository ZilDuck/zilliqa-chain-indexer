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

func CreateTransferAction(nft entity.Nft, blockNum uint64, txId string, prevOwner string) entity.NftAction {
	return entity.NftAction{
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     txId,
		BlockNum: blockNum,
		Action:   entity.TransferAction,
		From:     prevOwner,
		To:       nft.Owner,
	}
}

func CreateBurnAction(nft entity.Nft) entity.NftAction {
	return entity.NftAction{
		Contract: nft.Contract,
		TokenId:  nft.TokenId,
		TxID:     nft.TxID,
		BlockNum: nft.BlockNum,
		Action:   entity.BurnAction,
		From:     nft.Owner,
		To:       "",
	}
}
