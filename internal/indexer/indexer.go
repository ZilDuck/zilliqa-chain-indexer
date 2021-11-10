package indexer

import (
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/indexer/IndexOption"
	"github.com/dantudor/zil-indexer/internal/service/contract"
	"github.com/dantudor/zil-indexer/internal/service/nft"
	"github.com/dantudor/zil-indexer/internal/service/transaction"
	"go.uber.org/zap"
)

type Indexer interface {
	Index(option IndexOption.IndexOption, target uint64) error
}

type indexer struct {
	bulkIndexSize   uint64
	elastic         elastic_cache.Index
	txIndexer       transaction.Indexer
	txService       transaction.Service
	contractIndexer contract.Indexer
	nftIndexer      nft.Indexer
	rewinder        Rewinder
}

func NewIndexer(
	bulkIndexSize uint64,
	elastic elastic_cache.Index,
	txIndexer transaction.Indexer,
	txService transaction.Service,
	contractIndexer contract.Indexer,
	nftIndexer nft.Indexer,
	rewinder Rewinder,
) Indexer {
	return indexer{
		bulkIndexSize,
		elastic,
		txIndexer,
		txService,
		contractIndexer,
		nftIndexer,
		rewinder,
	}
}

func (i indexer) Index(option IndexOption.IndexOption, target uint64) error {
	lastBlockIndexed, err := i.txService.GetLastBlockNumIndexed()
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to get last block num from txs")
		lastBlockIndexed = 943800
	}
	lastBlockIndexed = 1294507

	height := lastBlockIndexed + 1
	if target != 0 && height == target {
		return nil
	}

	return i.index(height, target, option)
}

func (i indexer) index(height, target uint64, option IndexOption.IndexOption) error {
	size := uint64(1)
	if option == IndexOption.BatchIndex {
		if height > target {
			zap.L().With(zap.Uint64("target", target)).Info("Transactions indexed to target")
			return nil
		}

		size = i.bulkIndexSize
		if height+i.bulkIndexSize > target {
			size = target - height + 1
		}
	}

	txs, err := i.txIndexer.Index(height, size)
	if err != nil {
		zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index transactions")
		return err
	}

	if option == IndexOption.SingleIndex {
		_, err = i.contractIndexer.Index(txs)
		if err != nil {
			zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index Contacts")
			return err
		}

		err = i.nftIndexer.Index(txs)
		if err != nil {
			zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index NFTs")
			return err
		}
	}

	if option == IndexOption.BatchIndex {
		i.elastic.BatchPersist()
	} else {
		i.elastic.Persist()
	}

	if target != 0 && height > target {
		return nil
	}

	if option == IndexOption.BatchIndex {
		height = height + i.bulkIndexSize
	} else {
		height++
	}

	return i.index(height, target, option)
}
