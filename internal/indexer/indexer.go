package indexer

import (
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/indexer/IndexOption"
	"github.com/dantudor/zil-indexer/internal/repository"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"time"
)

type Indexer interface {
	Index(option IndexOption.IndexOption, target uint64) error
	RewindToHeight(blockNum uint64) error
	SetLastBlockNumIndexed(blockNum uint64)
	GetLastBlockNumIndexed() (uint64, error)
	ClearLastBlockNumIndexed()
}

type indexer struct {
	bulkIndexSize   uint64
	elastic         elastic_cache.Index
	txIndexer       TransactionIndexer
	contractIndexer ContractIndexer
	nftIndexer      NftIndexer
	txRepo          repository.TransactionRepository
	cache           *cache.Cache
}

func NewIndexer(
	bulkIndexSize uint64,
	elastic elastic_cache.Index,
	txIndexer TransactionIndexer,
	contractIndexer ContractIndexer,
	nftIndexer NftIndexer,
	txRepo repository.TransactionRepository,
	cache *cache.Cache,
) Indexer {
	return indexer{
		bulkIndexSize,
		elastic,
		txIndexer,
		contractIndexer,
		nftIndexer,
		txRepo,
		cache,
	}
}

func (i indexer) Index(option IndexOption.IndexOption, target uint64) error {
	lastBlockIndexed, err := i.GetLastBlockNumIndexed()
	if err != nil {
		if err.Error() == "best block not found" {
			lastBlockIndexed = 943800
		} else {
			time.Sleep(5 * time.Second)
			zap.L().With(zap.Error(err)).Fatal("Failed to get last block num from txs")
		}
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
		zap.L().With(zap.Error(err), zap.Uint64("height", height), zap.Uint64("size", size)).Warn("Failed to index transactions")
		return err
	}
	i.SetLastBlockNumIndexed(height + size - 1)

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

func (i indexer) RewindToHeight(blockNum uint64) error {
	zap.L().With(zap.Uint64("blockNum", blockNum)).Info("Rewinding to blockNum")

	i.elastic.ClearRequests()

	zap.L().With(zap.Uint64("blockNum", blockNum)).Info("Rewinding transaction index")

	if err := i.elastic.DeleteBlockNumGT(blockNum, elastic_cache.TransactionIndex.Get()); err != nil {
		return err
	}

	zap.L().With(zap.Uint64("blockNum", blockNum)).Info("Rewound to height")
	i.elastic.Persist()

	return nil
}

func (i indexer) ClearLastBlockNumIndexed() {
	i.cache.Delete("lastBlockNumIndexed")
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
