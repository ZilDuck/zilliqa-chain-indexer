package factory

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type ZilkroadMarketplaceFactory struct {
	nftRepo           repository.NftRepository
	contractRepo      repository.ContractRepository
	contractStateRepo repository.ContractStateRepository
}

func NewZilkroadMarketplaceFactory(nftRepo repository.NftRepository, contractRepo repository.ContractRepository, contractStateRepo repository.ContractStateRepository) ZilkroadMarketplaceFactory {
	return ZilkroadMarketplaceFactory{nftRepo, contractRepo,contractStateRepo}
}

func (f ZilkroadMarketplaceFactory) CreateListing(tx entity.Transaction) (*entity.MarketplaceListing, error) {
	listingEvent := tx.GetEventLogs(entity.MpZilkroadListingEvent)[0]

	tokenId, err := GetTokenId(listingEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get token id")
		return nil, err
	}

	contractAddr, err := listingEvent.Params.GetParam("nonfungible")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get contract addr")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(contractAddr.Value.String(), tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get nft")
		return nil, err
	}

	price, err := listingEvent.Params.GetParam("sell_price")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get sell_price")
		return nil, err
	}

	fungibleToken, err := listingEvent.Params.GetParam("fungible")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible token")
		return nil, err
	}

	fungibleContract, err := f.contractRepo.GetContractByAddress(fungibleToken.Value.String())
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.String("contractAddr", fungibleToken.Value.String()), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible contract")
		return nil, err
	}

	symbol, err := fungibleContract.Data.Params.GetParam("symbol")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible symbol")
		return nil, err
	}

	return &entity.MarketplaceListing{
		Marketplace: entity.ZilkroadMarketplace,
		Tx:          tx,
		Nft:         *nft,
		Cost:        price.Value.String(),
		Fungible:    symbol.Value.String(),
	}, nil
}

func (f ZilkroadMarketplaceFactory) CreateDelisting(tx entity.Transaction) (*entity.MarketplaceDelisting, error) {
	delistingEvent := tx.GetEventLogs(entity.MpZilkroadDelistingEvent)[0]
	tokenId, err := GetTokenId(delistingEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad delisting: Failed to get token id")
		return nil, err
	}

	contractAddr, err := delistingEvent.Params.GetParam("nonfungible")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad delisting: Failed to get contract addr")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(contractAddr.Value.String(), tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get nft")
		return nil, err
	}

	return &entity.MarketplaceDelisting{
		Marketplace: entity.ZilkroadMarketplace,
		Tx:          tx,
		Nft:         *nft,
	}, nil
}

func (f ZilkroadMarketplaceFactory) CreateSale(tx entity.Transaction) (*entity.MarketplaceSale, error) {
	salesEvent := tx.GetEventLogs(entity.MpArkySaleEvent)[0]

	buyer, err := salesEvent.Params.GetParam("buyer")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get buyer")
		return nil, err
	}

	seller, err := salesEvent.Params.GetParam("buyer")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get seller")
		return nil, err
	}

	tokenId, err := GetTokenId(salesEvent.Params)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get token id")
		return nil, err
	}

	contractAddr, err := salesEvent.Params.GetParam("nonfungible")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get contract addr")
		return nil, err
	}

	nft, err := f.nftRepo.GetNft(contractAddr.Value.String(), tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: NFT not found")
		return nil, err
	}

	costAsString, err := salesEvent.Params.GetParam("sell_price")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get sell_price")
		return nil, err
	}
	cost := costAsString.Value.String()

	royaltyAsString, err := salesEvent.Params.GetParam("royalty_amount")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get royalty_amount")
		return nil, err
	}
	royaltyFee := royaltyAsString.Value.String()
	royaltyFeeBps, err := f.getRoyaltyForContract(contractAddr.Value.String())
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get royalty fee bps from contract state")
		return nil, err
	}

	fungibleToken, err := salesEvent.Params.GetParam("fungible")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad sale: Failed to get fungible token")
		return nil, err
	}

	fungibleContract, err := f.contractRepo.GetContractByAddress(fungibleToken.Value.String())
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.String("contractAddr", fungibleToken.Value.String()), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible contract")
		return nil, err
	}

	symbol, err := fungibleContract.Data.Params.GetParam("symbol")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad sale: Failed to get fungible symbol")
		return nil, err
	}

	return &entity.MarketplaceSale{
		Marketplace:  entity.ArkyMarketplace,
		Tx:           tx,
		Nft:          *nft,
		Buyer:        buyer.Value.String(),
		Seller:       seller.Value.String(),
		Cost:         cost,
		Fee:          fmt.Sprintf("%d", entity.ZilkroadPlatformFee),
		Royalty:      royaltyFee,
		RoyaltyBps:   royaltyFeeBps,
		Fungible:     symbol.Value.String(),
	}, nil
}

func (f ZilkroadMarketplaceFactory) getRoyaltyForContract(contractAddr string) (string, error) {
	royaltyFeeBps, err := f.contractStateRepo.GetRoyaltyFeeBps(contractAddr)
	if err != nil {
		return "0", err
	}

	return fmt.Sprintf("%d", royaltyFeeBps), nil
}