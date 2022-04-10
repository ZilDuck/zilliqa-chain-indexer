package indexer

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"strconv"
)

type MarketplaceIndexer interface {
	IndexTxs(txs []entity.Transaction) error
}

type marketplaceIndexer struct {
	elastic      elastic_search.Index
	nftRepo      repository.NftRepository
	contractRepo repository.ContractRepository
}

func NewMarketplaceIndexer(
	elastic elastic_search.Index,
	nftRepo repository.NftRepository,
	contractRepo repository.ContractRepository,
) MarketplaceIndexer {
	return marketplaceIndexer{elastic, nftRepo, contractRepo}
}

func (i marketplaceIndexer) IndexTxs(txs []entity.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		if err := i.indexListings(tx); err != nil {
			return err
		}
		if err := i.indexDelistings(tx); err != nil {
			return err
		}
		if err := i.indexSales(tx); err != nil {
			return err
		}
	}

	return nil
}

func (i marketplaceIndexer) indexListings(tx entity.Transaction) error {
	if tx.IsMarketplaceListing(entity.OkimotoMarketplace) {
		listingEvent := tx.GetEventLogs(entity.MpOkiListingEvent)[0]
		tokenId, err := factory.GetTokenId(listingEvent.Params)
		if err != nil {
			zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get token id")
			return err
		}
		i.executeListing(entity.OkimotoMarketplace, tx, listingEvent.Address, tokenId, "", "")
		return nil
	}

	if tx.IsMarketplaceListing(entity.ZilkroadMarketplace) {
		for _, listingEvent := range tx.GetEventLogs(entity.MpZilkroadListingEvent) {
			tokenId, err := factory.GetTokenId(listingEvent.Params)
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get token id")
				return err
			}

			contractAddr, err := listingEvent.Params.GetParam("nonfungible")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get contract addr")
				return err
			}

			priceAsString, err := listingEvent.Params.GetParam("sell_price")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get sell_price")
				return err
			}
			price := priceAsString.Value.String()

			fungibleToken, err := listingEvent.Params.GetParam("fungible")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible token")
				return err
			}

			fungibleContract, err := i.contractRepo.GetContractByAddress(fungibleToken.Value.String())
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.String("contractAddr", fungibleToken.Value.String()), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible contract")
				return err
			}

			symbol, err := fungibleContract.Data.Params.GetParam("symbol")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible symbol")
				return err
			}

			i.executeListing(entity.ZilkroadMarketplace, tx, contractAddr.Value.String(), tokenId, price, symbol.Value.String())
		}
	}

	return nil
}

func (i marketplaceIndexer) indexDelistings(tx entity.Transaction) error {
	if tx.IsMarketplaceDelisting(entity.OkimotoMarketplace) {
		listingEvent := tx.GetEventLogs(entity.MpOkiDelistingEvent)[0]
		tokenId, err := factory.GetTokenId(listingEvent.Params)
		if err != nil {
			zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get token id")
			return err
		}
		i.executeDelisting(entity.OkimotoMarketplace, tx, listingEvent.Address, tokenId)
		return nil
	}

	if tx.IsMarketplaceDelisting(entity.ZilkroadMarketplace) {
		for _, delistingEvent := range tx.GetEventLogs(entity.MpZilkroadDelistingEvent) {
			tokenId, err := factory.GetTokenId(delistingEvent.Params)
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad delisting: Failed to get token id")
				return err
			}

			contractAddr, err := delistingEvent.Params.GetParam("nonfungible")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad delisting: Failed to get contract addr")
				return err
			}

			i.executeDelisting(entity.ZilkroadMarketplace, tx, contractAddr.Value.String(), tokenId)
		}
	}

	return nil
}

func (i marketplaceIndexer) indexSales(tx entity.Transaction) error {
	if tx.IsMarketplaceSale(entity.OkimotoMarketplace) {
		salesEvent := tx.GetEventLogs(entity.MpOkiSaleEvent)[0]

		tokenId, err := factory.GetTokenId(salesEvent.Params)
		if err != nil {
			zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get token id")
			return err
		}

		contractAddr := salesEvent.Address

		nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
		if err != nil {
			zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto listing: Failed to get nft")
			return err
		}

		buyer, err := salesEvent.Params.GetParam("recipient")
		if err != nil {
			zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Okimoto Sale: Failed to get buyer")
			return err
		}

		price := ""
		if tx.HasTransition("AddFunds") {
			addFunds := tx.GetTransition("AddFunds")[0]
			price = addFunds.Msg.Amount
		}

		i.executeSale(entity.OkimotoMarketplace, tx, salesEvent.Address, tokenId, buyer.Value.String(), nft.Owner, price, "", "", "ZIL")
		return nil
	}

	if tx.IsMarketplaceSale(entity.ZilkroadMarketplace) {
		for _, salesEvent := range tx.GetEventLogs(entity.MpZilkroadSaleEvent) {
			buyer, err := salesEvent.Params.GetParam("buyer")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get buyer")
				return err
			}
			seller, err := salesEvent.Params.GetParam("buyer")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get seller")
				return err
			}

			tokenId, err := factory.GetTokenId(salesEvent.Params)
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get token id")
				return err
			}

			contractAddr, err := salesEvent.Params.GetParam("nonfungible")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get contract addr")
				return err
			}

			priceAsString, err := salesEvent.Params.GetParam("sell_price")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get sell_price")
				return err
			}
			price := priceAsString.Value.String()

			royaltyAsString, err := salesEvent.Params.GetParam("royalty_amount")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get royalty_amount")
				return err
			}
			royalty := royaltyAsString.Value.String()

			fungibleToken, err := salesEvent.Params.GetParam("fungible")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad Sale: Failed to get fungible token")
				return err
			}

			fungibleContract, err := i.contractRepo.GetContractByAddress(fungibleToken.Value.String())
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.String("contractAddr", fungibleToken.Value.String()), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible contract")
				return err
			}

			symbol, err := fungibleContract.Data.Params.GetParam("symbol")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Zilkroad listing: Failed to get fungible symbol")
				return err
			}

			i.executeSale(entity.ZilkroadMarketplace, tx, contractAddr.Value.String(), tokenId, buyer.Value.String(), seller.Value.String(), price, "0", royalty, symbol.Value.String())
		}
	}

	if tx.IsMarketplaceSale(entity.ArkyMarketplace) {
		for _, tradeEvent := range tx.GetEventLogs(entity.MpArkySaleEvent) {
			token, err := tradeEvent.Params.GetParam("token")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get token")
				return err
			}

			var contractAddr string
			var tokenId int
			if token.Value != nil && len(token.Value.Arguments) == 2 {
				contractAddr = token.Value.Arguments[0].Primitive.(string)

				tokenIdString := token.Value.Arguments[1].Primitive.(string)
				tokenId, err = strconv.Atoi(tokenIdString)
				if err != nil {
					zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get token id")
					return err
				}
			}

			seller, err := tradeEvent.Params.GetParam("seller")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get seller")
				return err
			}

			buyer, err := tradeEvent.Params.GetParam("buyer")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get buyer")
				return err
			}

			proceeds, err := tradeEvent.Params.GetParam("proceeds")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get proceeds")
				return err
			}

			cost := "0"
			if proceeds.Value != nil && len(proceeds.Value.Arguments) == 2 {
				cost = proceeds.Value.Arguments[1].String()
			}

			fees, err := tradeEvent.Params.GetParam("fees")
			if err != nil {
				zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get proceeds")
				return err
			}

			fee := "0"
			if fees.Value != nil && len(fees.Value.Arguments) == 2 {
				fee = fees.Value.Arguments[1].String()
			}
			i.executeSale(entity.ArkyMarketplace, tx, contractAddr, uint64(tokenId), buyer.Value.Primitive.(string), seller.Value.Primitive.(string), cost, fee, "0", "")
		}
	}

	return nil
}

func (i marketplaceIndexer) executeListing(marketplace entity.Marketplace, tx entity.Transaction, contractAddr string, tokenId uint64, cost string, fungible string) {
	zap.L().With(
		zap.String("marketplace", string(marketplace)),
		zap.String("txId", tx.ID),
		zap.String("contractAddr", contractAddr),
		zap.Uint64("tokenId", tokenId),
		zap.String("fungible", fungible),
		zap.String("cost", cost),
	).Info("Marketplace listing")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().Error("Failed to find NFT")
		return
	}

	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMarketplaceListingAction(marketplace, *nft, tx.BlockNum, tx.ID, cost, fungible), elastic_search.NftAction)
}

func (i marketplaceIndexer) executeDelisting(marketplace entity.Marketplace, tx entity.Transaction, contractAddr string, tokenId uint64) {
	zap.L().With(
		zap.String("marketplace", string(marketplace)),
		zap.String("txId", tx.ID),
		zap.String("contractAddr", contractAddr),
		zap.Uint64("tokenId", tokenId),
	).Info("Marketplace delisting")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().Error("Failed to find NFT")
		return
	}

	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMarketplaceDelistingAction(marketplace, *nft, tx.BlockNum, tx.ID), elastic_search.NftAction)
}

func (i marketplaceIndexer) executeSale(marketplace entity.Marketplace, tx entity.Transaction, contractAddr string, tokenId uint64, buyer, seller string, cost, fee, royalty string, fungible string) {
	zap.L().With(
		zap.String("marketplace", string(marketplace)),
		zap.String("txId", tx.ID),
		zap.String("contractAddr", contractAddr),
		zap.Uint64("tokenId", tokenId),
		zap.String("from", seller),
		zap.String("to", buyer),
		zap.String("cost", cost),
		zap.String("fee", fee),
		zap.String("royalty", royalty),
		zap.String("fungible", fungible),
	).Info("Marketplace trade")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().Error("Failed to find NFT")
		return
	}

	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateTransferAction(*nft, tx.BlockNum, tx.ID, buyer, seller), elastic_search.Zrc6Transfer)
	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateMarketplaceSaleAction(marketplace, *nft, tx.BlockNum, tx.ID, buyer, seller, cost, fee, royalty, fungible), elastic_search.NftAction)
}
