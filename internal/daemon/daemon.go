package daemon

import (
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer/IndexOption"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type Daemon struct {
	elastic            elastic_search.Index
	firstBlockNum      uint64
	indexer            indexer.Indexer
	zilliqa            zilliqa.Service
	txRepo             repository.TransactionRepository
	nftRepo            repository.NftRepository
	contractRepo       repository.ContractRepository
	contractIndexer    indexer.ContractIndexer
	zrc1Indexer        indexer.Zrc1Indexer
	zrc6Indexer        indexer.Zrc6Indexer
	marketplaceIndexer indexer.MarketplaceIndexer
	metadataIndexer    indexer.MetadataIndexer
}

func NewDaemon(
	elastic elastic_search.Index,
	firstBlockNum uint64,
	indexer indexer.Indexer,
	zilliqa zilliqa.Service,
	txRepo repository.TransactionRepository,
	nftRepo repository.NftRepository,
	contractRepo repository.ContractRepository,
	contractIndexer indexer.ContractIndexer,
	zrc1Indexer indexer.Zrc1Indexer,
	zrc6Indexer indexer.Zrc6Indexer,
	marketplaceIndexer indexer.MarketplaceIndexer,
	metadataIndexer indexer.MetadataIndexer,
) *Daemon {
	return &Daemon{
		elastic,
		firstBlockNum,
		indexer,
		zilliqa,
		txRepo,
		nftRepo,
		contractRepo,
		contractIndexer,
		zrc1Indexer,
		zrc6Indexer,
		marketplaceIndexer,
		metadataIndexer,
	}
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
			d.indexer.SetLastBlockNumIndexed(d.firstBlockNum)
			return d.firstBlockNum
		}
		zap.L().With(zap.Error(err)).Fatal("Failed to find the best block num")
	}

	targetHeight := d.targetHeight(bestBlockNum)

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
	d.bulkIndexMarketPlaceSales(bestBlockNum)

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
	if err := d.contractIndexer.BulkIndex(d.bulkIndexContractsFrom(bestBlockNum)); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to bulk index contracts")
	}

	d.elastic.Persist()
	time.Sleep(2 * time.Second)
}

func (d *Daemon) bulkIndexContractsFrom(bestBlockNum uint64) uint64 {
	bulkIndexFrom := config.Get().BulkIndexContractsFrom
	if bulkIndexFrom == -1 {
		cBestBlockNum, err := d.contractRepo.GetBestBlockNum()
		if err == nil && cBestBlockNum < bestBlockNum {
			bulkIndexFrom = int(cBestBlockNum)
		} else {
			bulkIndexFrom = int(bestBlockNum)
		}
	}

	return uint64(bulkIndexFrom)
}

func (d *Daemon) bulkIndexNfts(bestBlockNum uint64) {
	bulkIndexNftsFrom := d.bulkIndexNftsFrom(bestBlockNum)
	size := 100
	contractPage := 1

	zap.L().With(zap.Uint64("bestBlockNum", bulkIndexNftsFrom)).Info("Bulk index NFTs")

	for {
		contracts, total, err := d.contractRepo.GetAllNftContracts(size, contractPage)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts when bulk indexing nfts")
			break
		}

		if contractPage == 1 {
			zap.S().Infof("Found %d nft contracts", total)
		}

		if len(contracts) == 0 {
			break
		}

		for _, c := range contracts {
			txPage := 1
			for {
				txs, total, err := d.txRepo.GetContractExecutionsByContractFrom(c, bulkIndexNftsFrom, size, txPage)
				if err != nil {
					zap.L().With(zap.Error(err)).Error("Failed to get txs when bulk indexing nfts")
				}
				if txPage == 1 && total != 0 {
					zap.S().Infof("Found %d nfts for contract %s", total, c.Address)
				}
				if len(txs) == 0 {
					break
				}

				for _, tx := range txs {
					if c.MatchesStandard(entity.ZRC1) {
						if err := d.zrc1Indexer.IndexTx(tx, c); err != nil {
							zap.L().With(zap.Error(err)).Error("Failed to bulk index Zrc1")
						}
					}
					if c.MatchesStandard(entity.ZRC6) {
						if err := d.zrc6Indexer.IndexTx(tx, c); err != nil {
							zap.L().With(zap.Error(err)).Error("Failed to bulk index Zrc6")
						}
					}
				}
				d.elastic.BatchPersist()
				txPage++
			}
		}
		d.elastic.BatchPersist()

		contractPage++
	}

	d.elastic.Persist()
	time.Sleep(2 * time.Second)
}

func (d *Daemon) bulkIndexMarketPlaceSales(bestBlockNum uint64) {
	bulkIndexFrom := d.bulkIndexNftsFrom(bestBlockNum)
	page := 1
	size := 100

	zap.L().With(zap.Uint64("bestBlockNum", bulkIndexFrom)).Info("Bulk index Marketplace sales")

	for {
		txs, _, err := d.txRepo.GetNftMarketplaceExecutionTxs(bulkIndexFrom, size, page)
		if err != nil {
			break
		}

		if len(txs) == 0 {
			break
		}
		d.marketplaceIndexer.IndexTxs(txs)
		d.elastic.BatchPersist()
		page++
	}
	d.elastic.Persist()
}

func (d *Daemon) bulkIndexNftsFrom(bestBlockNum uint64) uint64 {
	bulkIndexFrom := config.Get().BulkIndexNftsFrom
	if bulkIndexFrom == -1 {
		cBestBlockNum, err := d.nftRepo.GetBestBlockNum()
		if err == nil && cBestBlockNum < bestBlockNum {
			bulkIndexFrom = int(cBestBlockNum)
		} else {
			bulkIndexFrom = int(bestBlockNum)
		}
	}

	return uint64(bulkIndexFrom)
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
			}

			if err = d.indexer.Index(IndexOption.SingleIndex, targetHeight); err != nil {
				if !errors.Is(err, indexer.ErrBlockNotReady) {
					zap.L().With(zap.Error(err)).Fatal("Failed to index from subscriber")
				}
			}

			d.elastic.Persist()
		}

		time.Sleep(5 * time.Second)
	}
}

func (d Daemon) targetHeight(bestBlockNum uint64) uint64 {
	if config.Get().RewindToHeight != 0 {
		zap.L().With(zap.Uint64("height", config.Get().RewindToHeight)).Info("Rewinding to height from config")
		return config.Get().RewindToHeight
	}

	height := bestBlockNum

	if height >= config.Get().ReindexSize {
		return height - config.Get().ReindexSize
	}

	return d.firstBlockNum
}
