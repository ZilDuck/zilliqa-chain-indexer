package indexer

import (
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer/IndexOption"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"time"
)

type Indexer interface {
	Index(option IndexOption.IndexOption, target uint64) error

	SetLastBlockNumIndexed(blockNum uint64)
	GetLastBlockNumIndexed() (uint64, error)
}

type indexer struct {
	bulkIndexSize      uint64
	elastic            elastic_search.Index
	txIndexer          TransactionIndexer
	contractIndexer    ContractIndexer
	zrc1Indexer        Zrc1Indexer
	zrc6Indexer        Zrc6Indexer
	marketplaceIndexer MarketplaceIndexer
	txRepo             repository.TransactionRepository
	cache              *cache.Cache
}

var (
	ErrBlockNotReady = errors.New("tx block not ready")
)

func NewIndexer(
	bulkIndexSize uint64,
	elastic elastic_search.Index,
	txIndexer TransactionIndexer,
	contractIndexer ContractIndexer,
	zrc1Indexer Zrc1Indexer,
	zrc6Indexer Zrc6Indexer,
	marketplaceIndexer MarketplaceIndexer,
	txRepo repository.TransactionRepository,
	cache *cache.Cache,
) Indexer {
	return indexer{
		bulkIndexSize,
		elastic,
		txIndexer,
		contractIndexer,
		zrc1Indexer,
		zrc6Indexer,
		marketplaceIndexer,
		txRepo,
		cache,
	}
}

func (i indexer) Index(option IndexOption.IndexOption, target uint64) error {
	lastBlockIndexed, err := i.GetLastBlockNumIndexed()
	if err != nil {
		time.Sleep(5 * time.Second)
		zap.L().With(zap.Error(err)).Fatal("Failed to get last block num from txs")
	}

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
		zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Debug("Failed to index transactions")
		if err.Error()[:7] == "-32602:" || err.Error()[:4] == "-20:" {
			return ErrBlockNotReady
		}

		return err
	}
	i.SetLastBlockNumIndexed(height + size - 1)

	if option == IndexOption.SingleIndex {
		if err := i.contractIndexer.Index(txs); err != nil {
			zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index Contacts")
			return err
		}

		if err := i.zrc1Indexer.IndexTxs(txs); err != nil {
			zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index ZRC1s")
			return err
		}

		if err := i.zrc6Indexer.IndexTxs(txs); err != nil {
			zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index ZRC6s")
			return err
		}

		if err := i.marketplaceIndexer.IndexTxs(txs); err != nil {
			zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Error("Failed to index marketplace actions")
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

func (i indexer) SetLastBlockNumIndexed(blockNum uint64) {
	i.cache.Set("lastBlockNumIndexed", blockNum, cache.NoExpiration)
}

func (i indexer) GetLastBlockNumIndexed() (uint64, error) {
	if lastBlockNumIndexed, exists := i.cache.Get("lastBlockNumIndexed"); exists {
		blockNum := lastBlockNumIndexed.(uint64)
		return blockNum, nil
	}

	blockNum, err := i.txRepo.GetBestBlockNum()
	if err != nil {
		if err == repository.ErrBestBlockNumFound {
			return 0, err
		}
		zap.L().With(zap.Error(err)).Fatal("Failed to find the best block num")
	}
	i.SetLastBlockNumIndexed(blockNum)

	return blockNum, nil
}
