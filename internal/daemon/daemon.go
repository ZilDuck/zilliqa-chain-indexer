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

var (
	FirstContractBlockNum = uint64(943800)
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
			d.indexer.SetLastBlockNumIndexed(FirstContractBlockNum)
			return FirstContractBlockNum
		}
		zap.L().With(zap.Error(err)).Fatal("Failed to find the best block num")
	}

	targetHeight := targetHeight(bestBlockNum)

	d.indexer.SetLastBlockNumIndexed(targetHeight)

	zap.L().With(
		zap.Uint64("height", targetHeight),
	).Info("Best block")

	return targetHeight
}

func (d *Daemon) bulkIndex(bestBlockNum uint64) {
	if !config.Get().BulkIndex {
		return
	}

	zap.S().Infof("Bulk indexing from %d", bestBlockNum)

	d.bulkIndexTxs()
	d.bulkIndexContracts(bestBlockNum)
	d.bulkIndexNfts(bestBlockNum)

	zap.L().Info("Bulk indexing complete")
}

func (d *Daemon) getTargetHeight() uint64 {
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

	return targetHeight
}

func (d *Daemon) bulkIndexTxs() {
	if err := d.indexer.Index(IndexOption.BatchIndex, d.getTargetHeight()); err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to bulk index transactions")
	}

	d.elastic.Persist()
	time.Sleep(2 * time.Second)
}

func (d *Daemon) bulkIndexContracts(bestBlockNum uint64) {
	bulkIndexContractsFrom := config.Get().BulkIndexContractsFrom
	if bulkIndexContractsFrom == -1 {
		bulkIndexContractsFrom = int(bestBlockNum)
	}

	if err := d.contractIndexer.BulkIndex(uint64(bulkIndexContractsFrom)); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index contracts")
	}

	d.elastic.Persist()
	time.Sleep(2 * time.Second)
}

func (d *Daemon) bulkIndexNfts(bestBlockNum uint64) {
	BulkIndexNftsFrom := config.Get().BulkIndexNftsFrom
	if BulkIndexNftsFrom == -1 {
		BulkIndexNftsFrom = int(bestBlockNum)
	}

	if err := d.nftIndexer.BulkIndex(uint64(BulkIndexNftsFrom)); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index NFTs")
	}

	d.elastic.Persist()
	time.Sleep(2 * time.Second)
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

	return FirstContractBlockNum
}
