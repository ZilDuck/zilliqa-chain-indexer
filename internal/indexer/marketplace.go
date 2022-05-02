package indexer

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)

type MarketplaceIndexer interface {
	IndexTxs(txs []entity.Transaction) error
}

type marketplaceIndexer struct {
	elastic                    elastic_search.Index
	nftRepo                    repository.NftRepository
	contractRepo               repository.ContractRepository
	contractStateRepo          repository.ContractStateRepository
	zilkroadMarketplaceFactory factory.ZilkroadMarketplaceFactory
	okimotoMarketplaceFactory  factory.OkimotoMarketplaceFactory
	arkyMarketplaceFactory     factory.ArkyMarketplaceFactory
	mintableMarketplaceFactory factory.MintableMarketplaceFactory
}

func NewMarketplaceIndexer(
	elastic elastic_search.Index,
	nftRepo repository.NftRepository,
	contractRepo repository.ContractRepository,
	contractStateRepo repository.ContractStateRepository,
	zilkroadMarketplaceFactory factory.ZilkroadMarketplaceFactory,
	okimotoMarketplaceFactory factory.OkimotoMarketplaceFactory,
	arkyMarketplaceFactory factory.ArkyMarketplaceFactory,
	mintableMarketplaceFactory factory.MintableMarketplaceFactory,
) MarketplaceIndexer {
	return marketplaceIndexer{
		elastic,
		nftRepo,
		contractRepo,
		contractStateRepo,
		zilkroadMarketplaceFactory,
		okimotoMarketplaceFactory,
		arkyMarketplaceFactory,
		mintableMarketplaceFactory,
	}
}

func (i marketplaceIndexer) IndexTxs(txs []entity.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		if err := i.indexListings(tx); err != nil {
			continue
			//return err
		}
		if err := i.indexDelistings(tx); err != nil {
			continue
			//return err
		}
		if err := i.indexSales(tx); err != nil {
			continue
			//return err
		}
	}

	return nil
}

func (i marketplaceIndexer) indexListings(tx entity.Transaction) (err error) {
	var listing *entity.MarketplaceListing

	switch {
	case tx.IsMarketplaceListing(entity.OkimotoMarketplace):
		listing, err = i.okimotoMarketplaceFactory.CreateListing(tx)
	case tx.IsMarketplaceListing(entity.ZilkroadMarketplace):
		listing, err = i.zilkroadMarketplaceFactory.CreateListing(tx)
	case tx.IsMarketplaceListing(entity.MintableMarketplace):
		listing, err = i.mintableMarketplaceFactory.CreateListing(tx)
	}

	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Failed to create listing")
		return
	}

	if listing != nil {
		i.executeListing(*listing)
	}

	return
}

func (i marketplaceIndexer) indexDelistings(tx entity.Transaction) (err error) {
	var delisting *entity.MarketplaceDelisting

	switch {
	case tx.IsMarketplaceDelisting(entity.OkimotoMarketplace):
		delisting, err = i.okimotoMarketplaceFactory.CreateDelisting(tx)
	case tx.IsMarketplaceDelisting(entity.ZilkroadMarketplace):
		delisting, err = i.zilkroadMarketplaceFactory.CreateDelisting(tx)
	case tx.IsMarketplaceDelisting(entity.MintableMarketplace):
		delisting, err = i.mintableMarketplaceFactory.CreateDelisting(tx)
	}

	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Failed to create delisting")
		return err
	}

	if delisting != nil {
		i.executeDelisting(*delisting)
	}

	return nil
}

func (i marketplaceIndexer) indexSales(tx entity.Transaction) (err error) {
	var sale *entity.MarketplaceSale

	switch {
	case tx.IsMarketplaceSale(entity.ZilkroadMarketplace):
		sale, err = i.zilkroadMarketplaceFactory.CreateSale(tx)
	case tx.IsMarketplaceSale(entity.OkimotoMarketplace):
		sale, err = i.okimotoMarketplaceFactory.CreateSale(tx)
	case tx.IsMarketplaceSale(entity.ArkyMarketplace):
		sale, err = i.arkyMarketplaceFactory.CreateSale(tx)
	case tx.IsMarketplaceSale(entity.MintableMarketplace):
		sale, err = i.mintableMarketplaceFactory.CreateSale(tx)
	}

	if err != nil {
		zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Failed to create sale")
		return err
	}

	if sale != nil {
		i.executeSale(*sale)
	}

	return nil
}

func (i marketplaceIndexer) executeListing(listing entity.MarketplaceListing) {
	zap.L().With(
		zap.String("marketplace", string(listing.Marketplace)),
		zap.String("txId", listing.Tx.ID),
		zap.String("contract", listing.Nft.Contract),
		zap.Uint64("tokenId", listing.Nft.TokenId),
		zap.String("fungible", listing.Fungible),
		zap.String("cost", listing.Cost),
	).Info("Marketplace listing")


	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(),
		factory.CreateMarketplaceListingAction(listing.Marketplace, listing.Nft, listing.Tx.BlockNum, listing.Tx.ID,
			listing.Cost, listing.Fungible), elastic_search.NftAction)
}

func (i marketplaceIndexer) executeDelisting(delisting entity.MarketplaceDelisting) {
	zap.L().With(
		zap.String("marketplace", string(delisting.Marketplace)),
		zap.String("txId", delisting.Tx.ID),
		zap.String("contract", delisting.Nft.Contract),
		zap.Uint64("tokenId", delisting.Nft.TokenId),
	).Info("Marketplace delisting")

	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMarketplaceDelistingAction(delisting.Marketplace,
		delisting.Nft, delisting.Tx.BlockNum, delisting.Tx.ID), elastic_search.NftAction)
}

func (i marketplaceIndexer) executeSale(sale entity.MarketplaceSale) {
	zap.L().With(
		zap.String("marketplace", string(sale.Marketplace)),
		zap.String("txId", sale.Tx.ID),
		zap.String("contract", sale.Nft.Contract),
		zap.Uint64("tokenId", sale.Nft.TokenId),
		zap.String("from", sale.Seller),
		zap.String("to", sale.Buyer),
		zap.String("cost", sale.Cost),
		zap.String("fee", sale.Fee),
		zap.String("royalty", sale.Royalty),
		zap.String("royaltyBps", sale.RoyaltyBps),
		zap.String("fungible", sale.Fungible),
	).Info("Marketplace sale")

	sale.Nft.Owner = sale.Buyer
	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), sale.Nft, elastic_search.Zrc6Transfer)

	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateTransferAction(sale.Nft,
		sale.Tx.BlockNum, sale.Tx.ID, sale.Buyer, sale.Seller), elastic_search.Zrc6Transfer)
	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMarketplaceSaleAction(sale.Marketplace,
		sale.Nft, sale.Tx.BlockNum, sale.Tx.ID, sale.Buyer, sale.Seller, sale.Cost, sale.Fee, sale.Royalty,
		sale.RoyaltyBps, sale.Fungible), elastic_search.NftAction)
}