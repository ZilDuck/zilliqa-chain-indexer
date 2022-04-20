package factory

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"strconv"
)

type ArkyMarketplaceFactory struct {
	nftRepo           repository.NftRepository
	contractStateRepo repository.ContractStateRepository
}

func NewArkyMarketplaceFactory(nftRepo repository.NftRepository, contractStateRepo repository.ContractStateRepository) ArkyMarketplaceFactory {
	return ArkyMarketplaceFactory{nftRepo, contractStateRepo}
}

func (f ArkyMarketplaceFactory) CreateSale(tx entity.Transaction) (*entity.MarketplaceSale, error) {
	salesEvent := tx.GetEventLogs(entity.MpArkySaleEvent)[0]

	token, err := salesEvent.Params.GetParam("token")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get token")
		return nil, err
	}

	var contractAddr string
	var tokenId uint64
	if token.Value != nil && len(token.Value.Arguments) == 2 {
		contractAddr = token.Value.Arguments[0].Primitive.(string)
		tokenId, err = strconv.ParseUint(token.Value.Arguments[1].Primitive.(string), 10, 64)
		if err != nil {
			zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get token id")
			return nil, err
		}
	}

	nft, err := f.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: NFT not found")
		return nil, err
	}

	seller, err := salesEvent.Params.GetParam("seller")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get seller")
		return nil, err
	}

	buyer, err := salesEvent.Params.GetParam("buyer")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get buyer")
		return nil, err
	}

	proceeds, err := salesEvent.Params.GetParam("proceeds")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get proceeds")
		return nil, err
	}

	cost := "0"
	if proceeds.Value != nil && len(proceeds.Value.Arguments) == 2 {
		cost = proceeds.Value.Arguments[1].String()
	}

	fees, err := salesEvent.Params.GetParam("fees")
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get proceeds")
		return nil, err
	}

	fee := "0"
	if fees.Value != nil && len(fees.Value.Arguments) == 2 {
		fee = fees.Value.Arguments[1].String()
	}

	costInt, errCost := strconv.ParseUint(cost, 10, 64)
	feeInt, errFee := strconv.ParseUint(fee, 10, 64)
	if errCost == nil && errFee == nil {
		cost = fmt.Sprintf("%d", costInt + feeInt)
	}

	platformFee, royaltyFee, royaltyBps, err := f.getRoyaltyForContract(contractAddr, costInt, feeInt)
	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get royalty")
		return nil, err
	}

	return &entity.MarketplaceSale{
		Marketplace:  entity.ArkyMarketplace,
		Tx:           tx,
		Nft:          *nft,
		Buyer:        buyer.Value.String(),
		Seller:       seller.Value.String(),
		Cost:         cost,
		Fee:          platformFee,
		Royalty:      royaltyFee,
		RoyaltyBps:   royaltyBps,
		Fungible:     "ZIL",
	}, nil
}

func (f ArkyMarketplaceFactory) getRoyaltyForContract(contractAddr string, cost, totalFee uint64) (string, string, string, error) {
	cost = cost+totalFee
	totalFeePercent := uint((float64(totalFee)/float64(cost))*10000)
	platformFeePercent := entity.ArkyPlatformFee
	royaltyFeePercent := totalFeePercent - entity.ArkyPlatformFee

	royaltyFeeBps, err := f.contractStateRepo.GetRoyaltyFeeBps(contractAddr)
	if err != nil {
		return fmt.Sprintf("%d", totalFee), "0", "0", err
	}

	if royaltyFeeBps != royaltyFeePercent {
		zap.L().With(zap.Uint("royaltyFeeBps", royaltyFeeBps), zap.Uint("royaltyFeePercent", royaltyFeePercent)).
			Warn("royaltyFeeBps != royaltyFeePercent")
	}

	platformFee := uint64(float64(cost) * (float64(platformFeePercent)/10000))
	royaltyFee := uint64(float64(cost) * (float64(royaltyFeePercent)/10000))

	return fmt.Sprintf("%d", platformFee), fmt.Sprintf("%d", royaltyFee), fmt.Sprintf("%d", royaltyFeeBps), nil
}