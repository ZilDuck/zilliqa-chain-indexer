package factory

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"strconv"
)

type MintableMarketplaceFactory struct {
	nftRepo       repository.NftRepository
	nftActionRepo repository.NftActionRepository
}

func NewMintableMarketplaceFactory(nftRepo repository.NftRepository, nftActionRepo repository.NftActionRepository) MintableMarketplaceFactory {
	return MintableMarketplaceFactory{nftRepo, nftActionRepo}
}

func (f MintableMarketplaceFactory) CreateListing(tx entity.Transaction) (*entity.MarketplaceListing, error) {
	zap.L().Info("CreateListing: "+ tx.ID)
	listingEvent := tx.GetEventLogs(entity.MpMintableListingEvent)[0]

	orderInfo, err := listingEvent.Params.GetParam("order_info")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable listing: Failed to get order info")
	}

	arguments := orderInfo.Value.Arguments
	if len(arguments) < 4 {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable listing: Not enough arguments")
		return nil, err
	}

	tokenId, err := arguments[3].Uint64()
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable listing: Failed to get token id")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(arguments[2].String(), tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable listing: Failed to get nft")
		return nil, err
	}

	return &entity.MarketplaceListing{
		Marketplace: entity.MintableMarketplace,
		Tx:          tx,
		Nft:         *nft,
		Cost:        arguments[1].String(),
		Fungible:    "ZIL",
	}, nil
}

func (f MintableMarketplaceFactory) CreateDelisting(tx entity.Transaction) (*entity.MarketplaceDelisting, error) {
	zap.L().Info("CreateDelisting: "+ tx.ID)
	delistingEvents := tx.GetEventLogs("TransferSuccess")
	if len(delistingEvents) != 1 {
		zap.L().With(zap.String("txId", tx.ID)).Error("Mintable delisting: Failed to transfer event")
		return nil, errors.New("missing transfer success event")
	}

	delistingEvent := tx.GetEventLogs("TransferSuccess")[0]

	tokenId, err := GetTokenId(delistingEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable delisting: Failed to get token id")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(delistingEvent.Address, tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable delisting: Failed to get nft")
		return nil, err
	}

	return &entity.MarketplaceDelisting{
		Marketplace: entity.MintableMarketplace,
		Tx:          tx,
		Nft:         *nft,
	}, nil
}

func (f MintableMarketplaceFactory) CreateSale(tx entity.Transaction) (*entity.MarketplaceSale, error) {
	zap.L().Info("CreateSale: "+ tx.ID)
	salesEvent := tx.GetEventLogs(entity.MpMintableSaleEvent)[0]
	transferEvents := tx.GetEventLogs("TransferSuccess")
	if len(transferEvents) != 1 {
		zap.L().With(zap.String("txId", tx.ID)).Error("Mintable Sale: Failed to transfer event")
		return nil, errors.New("missing transfer success event")
	}

	transferEvent := tx.GetEventLogs("TransferSuccess")[0]

	tokenId, err := GetTokenId(salesEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable sale: Failed to get token id")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(transferEvent.Address, tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable sale: Failed to get nft")
		return nil, err
	}

	buyer, err := salesEvent.Params.GetParam("buyer_address")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable sale: Failed to get buyer")
		return nil, err
	}

	seller, err := f.nftActionRepo.GetNftOwnerBeforeBlockNum(*nft, tx.BlockNum)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable sale: Failed to get seller")
		return nil, err
	}

	totalCost := tx.Amount
	totalCostInt, errCost := strconv.ParseUint(totalCost, 10, 64)
	if errCost != nil{
		zap.L().With(zap.String("txId", tx.ID), zap.String("amount", totalCost), zap.Error(errCost)).Error("Mintable sale: Failed to get cost")
		return nil, err
	}

	costWOFee, err := salesEvent.Params.GetParam("price_sold")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable sale: Failed to get price sold")
		return nil, err
	}
	costWOFeeInt, err := costWOFee.Value.Uint64()
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Mintable sale: Failed to get cost w/o fee")
		return nil, err
	}

	fee := uint(0)
	if totalCostInt != 0 {
		fee = uint(((totalCostInt - costWOFeeInt) / totalCostInt) * 1000)
		if fee != entity.MintablePlatformFee {
			zap.L().With(zap.Uint("fee", fee), zap.Uint("MintablePlatformFee", entity.MintablePlatformFee)).
				Warn("royaltyFeeBps != royaltyFeePercent")
		}
	}

	return &entity.MarketplaceSale{
		Marketplace:  entity.MintableMarketplace,
		Tx:           tx,
		Nft:          *nft,
		Buyer:        buyer.Value.String(),
		Seller:       seller,
		Cost:         tx.Amount,
		Fee:          fmt.Sprintf("%d", fee),
		Royalty:      "0",
		RoyaltyBps:   "0",
		Fungible:     "ZIL",
	}, nil
}