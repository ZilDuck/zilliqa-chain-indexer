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
	IndexTx(tx entity.Transaction, c entity.Contract) error
	IndexContract(c entity.Contract) error
}

type marketplaceIndexer struct {
	elastic elastic_search.Index
	nftRepo repository.NftRepository
}

func NewMarketplaceIndexer(
	elastic elastic_search.Index,
	nftRepo repository.NftRepository,
) MarketplaceIndexer {
	return marketplaceIndexer{elastic, nftRepo}
}

func (i marketplaceIndexer) IndexTxs(txs []entity.Transaction) error {
	for _, tx := range txs {
		if !tx.IsContractExecution {
			continue
		}

		if tx.HasEventLog(entity.MpArkyTradeEvent) {
			for _, tradeEvent := range tx.GetEventLogs(entity.MpArkyTradeEvent) {
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

				cost := 0
				if proceeds.Value != nil && len(proceeds.Value.Arguments) == 2 {
					value := proceeds.Value.Arguments[1].Primitive.(string)
					if value != "" {
						cost, err = strconv.Atoi(value)
						if err != nil {
							zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get cost")
							return err
						}
					}
				}

				fees, err := tradeEvent.Params.GetParam("fees")
				if err != nil {
					zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get proceeds")
					return err
				}

				fee := 0
				if fees.Value != nil && len(fees.Value.Arguments) == 2 {
					value := fees.Value.Arguments[1].Primitive.(string)
					if value != "" {
						fee, err = strconv.Atoi(value)
						if err != nil {
							zap.L().With(zap.String("txId", tx.ID), zap.Error(err)).Error("Arky trade: Failed to get fees")
							return err
						}
					}
				}
				i.executeTrade(tx, contractAddr, uint64(tokenId), buyer.Value.Primitive.(string), seller.Value.Primitive.(string), cost+fee, fee)
			}
		}
	}

	return nil
}

func (i marketplaceIndexer) IndexTx(tx entity.Transaction, c entity.Contract) error {
	return nil
}

func (i marketplaceIndexer) IndexContract(c entity.Contract) error {

	return nil
}

func (i marketplaceIndexer) executeTrade(tx entity.Transaction, contractAddr string, tokenId uint64, buyer, seller string, cost, fee int) {
	zap.L().With(
		zap.Uint64("blockNum", tx.BlockNum),
		zap.String("contractAddr", contractAddr),
		zap.Uint64("tokenId", tokenId),
		zap.String("from", seller),
		zap.String("to", buyer),
		zap.Int("cost", cost),
		zap.Int("fee", fee),
	).Info("Executing trade")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().Error("Failed to find NFT")
		return
	}

	i.elastic.AddIndexRequest(elastic_search.NftActionIndex.Get(), factory.CreateSaleAction(*nft, tx.BlockNum, tx.ID, buyer, seller, cost, fee), elastic_search.NftAction)
}

