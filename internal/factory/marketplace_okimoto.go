package factory

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type OkimotoMarketplaceFactory struct {
	nftRepo       repository.NftRepository
	nftActionRepo repository.NftActionRepository
}

func NewOkimotoMarketplaceFactory(nftRepo repository.NftRepository, nftActionRepo repository.NftActionRepository) OkimotoMarketplaceFactory {
	return OkimotoMarketplaceFactory{nftRepo, nftActionRepo}
}

func (f OkimotoMarketplaceFactory) CreateListing(tx entity.Transaction) (*entity.MarketplaceListing, error) {
	listingEvent := tx.GetEventLogs(entity.MpOkiListingEvent)[0]
	tokenId, err := GetTokenId(listingEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get token id")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(listingEvent.Address, tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get nft")
		return nil, err
	}

	return &entity.MarketplaceListing{
		Marketplace: entity.OkimotoMarketplace,
		Tx:          tx,
		Nft:         *nft,
		Cost:        "",
		Fungible:    "ZIL",
	}, nil
}

func (f OkimotoMarketplaceFactory) CreateDelisting(tx entity.Transaction) (*entity.MarketplaceDelisting, error) {
	delistingEvent := tx.GetEventLogs(entity.MpOkiDelistingEvent)[0]
	tokenId, err := GetTokenId(delistingEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto delisting: Failed to get token id")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(delistingEvent.Address, tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto delisting: Failed to get nft")
		return nil, err
	}

	return &entity.MarketplaceDelisting{
		Marketplace: entity.OkimotoMarketplace,
		Tx:          tx,
		Nft:         *nft,
	}, nil
}

func (f OkimotoMarketplaceFactory) CreateSale(tx entity.Transaction) (*entity.MarketplaceSale, error) {
	salesEvent := tx.GetEventLogs(entity.MpOkiSaleEvent)[0]

	tokenId, err := GetTokenId(salesEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto sale: Failed to get token id")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(salesEvent.Address, tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto sale: Failed to get nft")
		return nil, err
	}

	buyer, err := salesEvent.Params.GetParam("recipient")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto sale: Failed to get buyer")
		return nil, err
	}

	seller, err := f.nftActionRepo.GetNftOwnerBeforeBlockNum(*nft, tx.BlockNum)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto sale: Failed to get seller")
		return nil, err
	}

	cost := ""
	if tx.HasTransition("AddFunds") {
		addFunds := tx.GetTransition("AddFunds")[0]
		cost = addFunds.Msg.Amount
	}

	return &entity.MarketplaceSale{
		Marketplace:  entity.OkimotoMarketplace,
		Tx:           tx,
		Nft:          *nft,
		Buyer:        buyer.Value.String(),
		Seller:       seller,
		Cost:         cost,
		Fee:          fmt.Sprintf("%d", entity.OkimotoPlatformFee),
		Royalty:      "0",
		RoyaltyBps:   "0",
		Fungible:     "ZIL",
	}, nil
}