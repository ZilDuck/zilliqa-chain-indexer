package daemon

import (
	"github.com/dantudor/zil-indexer/generated/dic"
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/dantudor/zil-indexer/internal/indexer/IndexOption"
	"github.com/sarulabs/dingo/v3"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var container *dic.Container

func Execute() {
	initialize()

	container.GetElastic().InstallMappings()

	if config.Get().Reindex == true {
		zap.L().Info("Reindex complete")
		return
	}

	bestBlock := rewind()

	if config.Get().BulkIndex == true {
		bulkIndex(bestBlock)
		zap.L().Info("Bulk indexing complete")
	}

	if config.Get().Subscribe == true {
		subscribe()
	}
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer(dingo.App)
	zap.L().Info("Indexer Started")
}

func rewind() uint64 {
	bestBlockNum, err := container.GetTxRepo().GetBestBlockNum()
	if err != nil {
		return 0
	}

	target := targetHeight(bestBlockNum)

	zap.L().Info("Rewind Index", zap.Uint64("from", bestBlockNum), zap.Uint64("to", target))
	if err := container.GetRewinder().RewindToHeight(target); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to rewind index")
		time.Sleep(5 * time.Second)
		return rewind()
	}

	container.GetElastic().Persist()
	zap.L().Info("Sleep for 5 seconds")
	time.Sleep(5 * time.Second)

	bestBlockNum, err = container.GetTxRepo().GetBestBlockNum()
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to get best block")
	}

	if target != bestBlockNum {
		return rewind()
	}

	container.GetTxService().SetLastBlockNumIndexed(bestBlockNum)

	zap.L().With(
		zap.Uint64("height", bestBlockNum),
	).Info("Best block")

	return bestBlockNum
}

func bulkIndex(bestBlock uint64) {
	zap.S().Infof("Bulk indexing from %d", bestBlock)

	targetHeight := config.Get().BulkTargetHeight
	if targetHeight == 0 {
		latestCoreTxBlock, err := container.GetZilliqa().GetLatestTxBlock()
		if err != nil {
			zap.L().With(zap.Error(err)).Fatal("Failed to get latest block from zilliqa")
		}
		targetHeight, err = strconv.ParseUint(latestCoreTxBlock.Header.BlockNum, 0, 64)
		if err != nil {
			zap.L().With(zap.Error(err)).Fatal("Failed to parse latest block num")
		}
	}
	zap.S().Infof("Target Height: %d", targetHeight)

	if err := container.GetIndexer().Index(IndexOption.BatchIndex, targetHeight); err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to bulk index transactions")
	}
	container.GetElastic().Persist()
	time.Sleep(2 * time.Second)

	if err := container.GetContractIndexer().BulkIndex(bestBlock); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index contracts")
	}
	container.GetElastic().Persist()
	time.Sleep(2 * time.Second)

	if err := container.GetNftIndexer().BulkIndex(bestBlock); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index NFTs")
	}
	container.GetElastic().Persist()
	time.Sleep(2 * time.Second)
}

func subscribe() {
	zap.L().Info("Starting subscriber")
	for {
		latestCoreTxBlock, err := container.GetZilliqa().GetLatestTxBlock()
		if err == nil {
			targetHeight, err := strconv.ParseUint(latestCoreTxBlock.Header.BlockNum, 0, 64)
			if err != nil {
				zap.L().With(zap.Error(err)).Fatal("Failed to parse latest block num")
			} else {
				err = container.GetIndexer().Index(IndexOption.SingleIndex, targetHeight)
			}
			if err != nil {
				container.GetElastic().Persist()
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
