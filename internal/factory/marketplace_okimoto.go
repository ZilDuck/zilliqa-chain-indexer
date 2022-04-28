package factory

import (
	"encoding/json"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type OkimotoMarketplaceFactory struct {
	nftRepo           repository.NftRepository
	nftActionRepo     repository.NftActionRepository
	contractStateRepo repository.ContractStateRepository
}

func NewOkimotoMarketplaceFactory(nftRepo repository.NftRepository, nftActionRepo repository.NftActionRepository, contractStateRepo repository.ContractStateRepository) OkimotoMarketplaceFactory {
	return OkimotoMarketplaceFactory{nftRepo, nftActionRepo, contractStateRepo}
}

func (f OkimotoMarketplaceFactory) CreateListing(tx entity.Transaction) (*entity.MarketplaceListing, error) {
	listingId, err := tx.Data.Params.GetParam("listing_id")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get listing id")
		return nil, err
	}

	_, err = tx.Data.Params.GetParam("desired_pay")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get desired pay")
		return nil, err
	}

	state, err := f.contractStateRepo.GetStateByAddress(tx.ContractAddress)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get state")
		return nil, err
	}

	listingString, ok := state.GetElement("listings")
	if !ok {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get contract listings")
		return nil, err
	}

	var listings map[string]interface{}
	err = json.Unmarshal([]byte(listingString), &listings)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to decode listings")
		return nil, err
	}

	listingInterface, ok := listings[listingId.Value.String()]
	if !ok {
		zap.L().With(zap.String("txId", tx.ID), zap.String("listingId", listingId.Value.String()), zap.Error(err)).Error("Okimoto listing: Failed to find listing")
		return nil, err
	}

	listing := CreateValueObject(listingInterface)
	if listing == nil || len(listing.Arguments) < 5 {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get listing arguments")
		return nil, err
	}

	tokenId, err := listing.Arguments[2].Uint64()
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get token id")
		return nil, err
	}

	costArgs := listing.Arguments[4].Arguments
	if len(costArgs) != 1 {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get cost")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(listing.Arguments[1].String(), tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get nft")
		return nil, err
	}

	return &entity.MarketplaceListing{
		Marketplace: entity.OkimotoMarketplace,
		Tx:          tx,
		Nft:         *nft,
		Cost:        costArgs[0].String(),
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