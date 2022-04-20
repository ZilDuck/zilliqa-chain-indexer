package factory

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"math/big"
)

type MintableMarketplaceFactory struct {
	nftRepo       repository.NftRepository
	nftActionRepo repository.NftActionRepository
}

func NewMintableMarketplaceFactory(nftRepo repository.NftRepository, nftActionRepo repository.NftActionRepository) MintableMarketplaceFactory {
	return MintableMarketplaceFactory{nftRepo, nftActionRepo}
}

func (f MintableMarketplaceFactory) CreateListing(tx entity.Transaction) (*entity.MarketplaceListing, error) {
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
	salesEvent := tx.GetEventLogs(entity.MpMintableSaleEvent)[0]
	transferEvents := tx.GetEventLogs("TransferSuccess")
	if len(transferEvents) != 1 {
		zap.L().With(zap.String("txId", tx.ID)).Error("Mintable Sale: Failed to transfer event")
		return nil, errors.New("missing transfer success event")
	}

	transferEvent := tx.GetEventLogs("TransferSuccess")[0]

	tokenId, err := GetTokenId(salesEvent.Params)
	if err != nil {
		zap.L().With(
			zap.String("txId", tx.ID),
			zap.String("contract", transferEvent.Address),
			zap.Uint64("tokenId", tokenId),
			zap.Error(err),
		).Error("Mintable sale: Failed to get tokenId")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(transferEvent.Address, tokenId)
	if err != nil {
		zap.L().With(
			zap.String("txId", tx.ID),
			zap.String("contract", transferEvent.Address),
			zap.Uint64("tokenId", tokenId),
			zap.Error(err),
		).Error("Mintable sale: Failed to get nft")
		return nil, err
	}

	buyer, err := salesEvent.Params.GetParam("buyer_address")
	if err != nil {
		zap.L().With(
			zap.String("txId", tx.ID),
			zap.String("contract", transferEvent.Address),
			zap.Uint64("tokenId", tokenId),
			zap.Error(err),
		).Error("Mintable sale: Failed to get buyer")
		return nil, err
	}

	seller, err := f.nftActionRepo.GetNftOwnerBeforeBlockNum(*nft, tx.BlockNum)
	if err != nil {
		zap.L().With(
			zap.String("txId", tx.ID),
			zap.String("contract", transferEvent.Address),
			zap.Uint64("tokenId", tokenId),
			zap.Error(err),
		).Error("Mintable sale: Failed to get seller")
		return nil, err
	}

	costWOFee, err := salesEvent.Params.GetParam("price_sold")
	if err != nil {
		zap.L().With(
			zap.String("txId", tx.ID),
			zap.String("contract", transferEvent.Address),
			zap.Uint64("tokenId", tokenId),
			zap.Error(err),
		).Error("Mintable sale: Failed to get price sold")
		return nil, err
	}

	fee, err := calculateFees(tx.Amount, costWOFee.Value.String())
	if err != nil {
		zap.L().With(
			zap.String("txId", tx.ID),
			zap.String("contract", transferEvent.Address),
			zap.Uint64("tokenId", tokenId),
			zap.Error(err),
		).Error("Mintable sale: Failed to get fee")
		return nil, err
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

func calculateFees(totalCost, costWOFee string) (string, error) {
	bigTotal, ok := new(big.Int).SetString(totalCost, 10)
	if !ok {
		return "", fmt.Errorf("invalid total cost (%s)", totalCost)
	}

	bigCostWOFee, ok := new(big.Int).SetString(costWOFee, 10)
	if !ok {
		return "", fmt.Errorf("invalid cost without fee (%s)", costWOFee)
	}

	fee := big.NewInt(0).Sub(bigTotal, bigCostWOFee).String()

	return fee, nil
}