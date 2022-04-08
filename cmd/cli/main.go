package main

import (
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"go.uber.org/zap"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

var container *dic.Container

func main() {
	config.Init()
	container, _ = dic.NewContainer()

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

func processMetadata(c *cli.Context) error {
	var status entity.MetadataStatus


	size, err := container.GetMessenger().GetQueueSize(messenger.MetadataRefresh)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("Could not get the queue size")
		return nil
	}
	if *size != 0 {
		zap.S().Errorf("Can only schedule metadata updates when the queue is empty, current size (%d)", size)
		return nil
	}

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

	if err := container.GetMetadataIndexer().RefreshByStatus(status, metadataError); err != nil {
		zap.S().With(zap.Error(err)).Fatalf("Failed to process %s metadata", status)
		return err
	}
	zap.L().Info("Metadata processing complete")

	return nil
}

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
		container.GetMarketplaceIndexer().IndexTxs(txs)
		container.GetElastic().BatchPersist()
		page++
	}
	container.GetElastic().Persist()

	return nil
}