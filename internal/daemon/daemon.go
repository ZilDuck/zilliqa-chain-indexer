package daemon

import (
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/indexer"
	"github.com/dantudor/zil-indexer/internal/indexer/IndexOption"
	"github.com/dantudor/zil-indexer/internal/repository"
	"github.com/dantudor/zil-indexer/internal/zilliqa"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type Daemon struct {
	elastic         elastic_cache.Index
	indexer         indexer.Indexer
	zilliqa         zilliqa.Service
	txRepo          repository.TransactionRepository
	contractIndexer indexer.ContractIndexer
	nftIndexer      indexer.NftIndexer
}

func NewDaemon(
	elastic elastic_cache.Index,
	indexer indexer.Indexer,
	zilliqa zilliqa.Service,
	txRepo repository.TransactionRepository,
	contractIndexer indexer.ContractIndexer,
	nftIndexer indexer.NftIndexer,
) *Daemon {
	return &Daemon{elastic, indexer, zilliqa, txRepo, contractIndexer, nftIndexer}
}

func (d *Daemon) Execute() {
	d.elastic.InstallMappings()

	if config.Get().Reindex == true {
		zap.L().Info("Reindex complete")
		return
	}

	bestBlock := d.rewind()
	d.bulkIndex(bestBlock)
	d.subscribe()
}

func (d *Daemon) rewind() uint64 {
	bestBlockNum, err := d.txRepo.GetBestBlockNum()
	if err != nil {
		if err == repository.ErrBestBlockNumFound {
			return 0
		}
		zap.L().With(zap.Error(err)).Fatal("Failed to find the best block num")
	}

	target := targetHeight(bestBlockNum)

	zap.L().Info("Rewind Index", zap.Uint64("from", bestBlockNum), zap.Uint64("to", target))
	if err := d.indexer.RewindToHeight(target); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to rewind index")
		time.Sleep(2 * time.Second)

		return d.rewind()
	}

	d.elastic.Persist()
	zap.L().Info("Sleep for 5 seconds")
	time.Sleep(5 * time.Second)

	bestBlockNum, err = d.txRepo.GetBestBlockNum()
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to find the best block num")
	}

	if target != bestBlockNum {
		return d.rewind()
	}

	d.indexer.SetLastBlockNumIndexed(bestBlockNum)

	zap.L().With(
		zap.Uint64("height", bestBlockNum),
	).Info("Best block")

	return bestBlockNum
}

func (d *Daemon) bulkIndex(bestBlock uint64) {
	if !config.Get().BulkIndex {
		return
	}

	zap.S().Infof("Bulk indexing from %d", bestBlock)

	targetHeight := config.Get().BulkTargetHeight
	if targetHeight == 0 {
		latestCoreTxBlock, err := d.zilliqa.GetLatestTxBlock()
		if err != nil {
			zap.L().With(zap.Error(err)).Fatal("Failed to get latest block from zilliqa")
		}
		targetHeight, err = strconv.ParseUint(latestCoreTxBlock.Header.BlockNum, 0, 64)
		if err != nil {
			zap.L().With(zap.Error(err)).Fatal("Failed to parse latest block num")
		}
	}
	zap.S().Infof("Target Height: %d", targetHeight)

	if err := d.indexer.Index(IndexOption.BatchIndex, targetHeight); err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to bulk index transactions")
	}
	d.elastic.Persist()
	time.Sleep(2 * time.Second)

	if err := d.contractIndexer.BulkIndex(bestBlock); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index contracts")
	}
	d.elastic.Persist()
	time.Sleep(2 * time.Second)

	if err := d.nftIndexer.BulkIndex(bestBlock); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index NFTs")
	}
	d.elastic.Persist()
	time.Sleep(2 * time.Second)

	zap.L().Info("Bulk indexing complete")
}

func (d *Daemon) subscribe() {
	if !config.Get().Subscribe {
		return
	}

	zap.L().Info("Starting subscriber")
	for {
		latestCoreTxBlock, err := d.zilliqa.GetLatestTxBlock()
		if err == nil {
			targetHeight, err := strconv.ParseUint(latestCoreTxBlock.Header.BlockNum, 0, 64)
			if err != nil {
				zap.L().With(zap.Error(err)).Fatal("Failed to parse latest block num")
			} else {
				err = d.indexer.Index(IndexOption.SingleIndex, targetHeight)
			}
			if err != nil {
				d.elastic.Persist()
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func targetHeight(bestBlockNum uint64) uint64 {
	if config.Get().RewindToHeight != 0 {
		zap.L().With(zap.Uint64("height", config.Get().RewindToHeight)).Info("Rewinding to height from config")
		return config.Get().RewindToHeight
	}

	height := bestBlockNum

	if height >= config.Get().ReindexSize {
		return height - config.Get().ReindexSize
	}

	return 0
}
