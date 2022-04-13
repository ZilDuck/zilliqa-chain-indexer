package main

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/indexer"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"os"
	"strings"
)

var (
	container          *dic.Container
	elastic            elastic_search.Index
	contractRepo       repository.ContractRepository
	txRepo             repository.TransactionRepository
	nftRepo            repository.NftRepository
	marketplaceIndexer indexer.MarketplaceIndexer
	metadataIndexer    indexer.MetadataIndexer
	zrc1Indexer        indexer.Zrc1Indexer
	zrc6Indexer        indexer.Zrc6Indexer
	messengerService   messenger.MessageService
)


func main() {
	config.Init("cli")

	container, _ = dic.NewContainer()
	elastic = container.GetElastic()
	contractRepo = container.GetContractRepo()
	txRepo = container.GetTxRepo()
	nftRepo = container.GetNftRepo()
	marketplaceIndexer = container.GetMarketplaceIndexer()
	metadataIndexer = container.GetMetadataIndexer()
	zrc1Indexer = container.GetZrc1Indexer()
	zrc6Indexer = container.GetZrc6Indexer()
	messengerService = container.GetMessenger()

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "metadata",
				Usage:   "queue NFTs for metadata refresh by their status (pending, failed, or success)",
				Action:  processMetadata,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "err", Value: "", Usage: "filter NFTs by metadata error"},
				},
			},
			{
				Name:    "importNfts",
				Usage:   "Import NFTs",
				Action:  importNfts,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "contract", Value: "", Usage: "Import for a single contract"},
					&cli.StringFlag{Name: "purge", Value: "false", Usage: "Purge the contract"},
				},
			},
			{
				Name:    "marketplace",
				Usage:   "Reindex all marketplace actions",
				Action:  processMarketplaceActions,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to start CLI")
	}
}




// METADATA
func processMetadata(c *cli.Context) error {
	size, err := messengerService.GetQueueSize(messenger.MetadataRefresh)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Could not get the queue size")
		return nil
	}
	if *size != 0 {
		zap.S().Errorf("Can only schedule metadata updates when the queue is empty, current size (%d)", *size)
		return nil
	}

	var status entity.MetadataStatus
	switch strings.ToLower(c.Args().First()) {
	case "pending":
		status = entity.MetadataPending
	case "failed":
		status = entity.MetadataFailure
	case "success":
		status = entity.MetadataSuccess
	default:
		zap.L().Error("No status provided")
		return nil
	}
	metadataError := c.Args().Get(1)

	zap.S().Infof("Processing Metadata: %s, %s", status, metadataError)

	if err := metadataIndexer.RefreshByStatus(status, metadataError); err != nil {
		zap.S().With(zap.Error(err)).Fatalf("Failed to process %s metadata", status)
		return err
	}
	zap.L().Info("Metadata processing complete")

	return nil
}




// NFTS
func importNfts(c *cli.Context) error {
	contractAddr := c.String("contract")
	purge := c.Bool("purge")

	if contractAddr != "" {
		contract, err := contractRepo.GetContractByAddress(contractAddr)
		if err != nil {
			zap.S().Errorf("Failed to find contract: %s", contractAddr)
			return err
		}

		if purge {
			if err := nftRepo.PurgeContract(contract.Address); err != nil {
				return err
			}
		}

		importNftsForContract(*contract)
		importMarketplaceSalesForContract(*contract)
	} else {
		importAllNfts()
		importMarketplaceSales()
	}

	zap.L().Info("Ready for exit")
	fmt.Scanln()

	return nil
}

func importAllNfts() {
	size := 100
	page := 1

	for {
		contracts, total, err := contractRepo.GetAllNftContracts(size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contracts")
			break
		}
		if page == 1 {
			zap.S().Infof("Found %d Contracts", total)
		}
		if len(contracts) == 0 {
			break
		}
		for _, c := range contracts {
			importNftsForContract(c)
		}
		elastic.BatchPersist()
		page++
	}
	elastic.Persist()
}

func importNftsForContract(contract entity.Contract) {
	zap.L().Info("*** Import Nfts For Contract: "+contract.Address)
	_ = nftRepo.PurgeActions(contract.Address)

	if contract.MatchesStandard(entity.ZRC6) {
		zap.L().With(zap.String("contractAddr", contract.Address), zap.String("shape", "ZRC6")).Info("Import nfts for contract")
		if err := zrc6Indexer.IndexContract(contract); err != nil {
			zap.S().Fatalf("Failed to index ZRC6 NFTs for contract %s", contract.Address)
		}
	} else if contract.MatchesStandard(entity.ZRC1) {
		zap.L().With(zap.String("contractAddr", contract.Address), zap.String("shape", "ZRC1")).Info("Import nfts for contract")
		if err := zrc1Indexer.IndexContract(contract); err != nil {
			zap.S().Fatalf("Failed to index ZRC1 NFTs for contract %s", contract.Address)
		}
	}
	elastic.Persist()
}

func importMarketplaceSales() {
	page := 1
	size := 100
	for {
		txs, _, err := txRepo.GetNftMarketplaceExecutionTxs(0, size, page)
		if err != nil {
			break
		}

		if len(txs) == 0 {
			break
		}
		_ = marketplaceIndexer.IndexTxs(txs)
		elastic.BatchPersist()
		page++
	}
	elastic.Persist()
}

func importMarketplaceSalesForContract(c entity.Contract) {
	page := 1
	size := 100
	for {
		txs, _, err := txRepo.GetContractExecutionsByContract(c, size, page)
		if err != nil {
			break
		}

		if len(txs) == 0 {
			break
		}
		_ = marketplaceIndexer.IndexTxs(txs)
		elastic.BatchPersist()
		page++
	}
	elastic.Persist()
}





// MARKETPLACE
func processMarketplaceActions(c *cli.Context) error {
	page := 1
	size := 100
	for {
		txs, _, err := container.GetTxRepo().GetNftMarketplaceExecutionTxs(0, size, page)
		if err != nil {
			return err
		}

		if len(txs) == 0 {
			break
		}
		marketplaceIndexer.IndexTxs(txs)
		elastic.BatchPersist()
		page++
	}
	elastic.Persist()

	return nil
}
