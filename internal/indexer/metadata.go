package indexer

import (
	"encoding/json"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/event"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"time"
)

type MetadataIndexer interface {
	TriggerMetadataRefresh(el interface{})
	RefreshMetadata(contractAddr string, tokenId uint64) (*entity.Nft, error)
	RefreshByStatus(status entity.MetadataStatus, metadataError string) error
}

type metadataIndexer struct {
	elastic         elastic_search.Index
	nftRepo         repository.NftRepository
	messageService  messenger.MessageService
	metadataService metadata.Service
}

func NewMetadataIndexer(
	elastic elastic_search.Index,
	nftRepo repository.NftRepository,
	messageService messenger.MessageService,
	metadataService metadata.Service,
) MetadataIndexer {
	i := metadataIndexer{elastic, nftRepo, messageService, metadataService}

	event.AddEventListener(event.NftMintedEvent, i.TriggerMetadataRefresh)
	event.AddEventListener(event.ContractBaseUriUpdatedEvent, i.TriggerMetadataRefresh)
	event.AddEventListener(event.TokenUriUpdatedEvent, i.TriggerMetadataRefresh)

	return i
}

func (i metadataIndexer) TriggerMetadataRefresh(el interface{}) {
	if !config.Get().EventsSupported {
		zap.L().Warn("TriggerMetadataRefresh: Events not supported")
		return
	}
	nft := el.(entity.Nft)

	_, exists := config.Get().ContractsWithoutMetadata[nft.Contract]
	if exists {
		zap.L().With(zap.String("contract", nft.Contract)).Warn("Contract doesnt support metadata")
		return
	}

	if nft.Metadata == nil || nft.Metadata.UriEmpty() {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.MetadataRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue metadata refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger MetaData Refresh")
}

func (i metadataIndexer) RefreshMetadata(contractAddr string, tokenId uint64) (*entity.Nft, error) {
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId)).Info("NFT Refresh Metadata")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		return nil, err
	}

	properties, err := i.metadataService.FetchMetadata(*nft)
	if err != nil {
		if err == metadata.ErrNoSuchHost ||
			err == metadata.ErrNotFound ||
			err == metadata.ErrBadRequest ||
			err == metadata.ErrInvalidContent ||
			err == metadata.ErrUnsupportedProtocolScheme ||
			err == metadata.ErrMetadataNotFound ||
			err == metadata.ErrTimeout {
			if len(nft.Metadata.Properties) == 0 {
				nft.Metadata.Status = entity.MetadataFailure
			}
			nft.Metadata.Error = err.Error()
		} else {
			nft.Metadata.Status = entity.MetadataFailure
			nft.Metadata.Error = fmt.Sprintf("Unexpected: %s", err.Error())
		}
		nft.Metadata.Attempts++
		nft.Metadata.UpdatedAt = time.Now()
		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
		i.elastic.BatchPersist()

		return nil, err
	}

	nft.Metadata.Properties = properties
	nft.Metadata.Error = ""
	nft.Metadata.UpdatedAt = time.Now()
	nft.Metadata.Status = entity.MetadataSuccess
	if assetUri, err := nft.Metadata.GetAssetUri(); err == nil {
		nft.AssetUri = assetUri
	}

	if err := i.nftRepo.ResetMetadata(*nft); err != nil {
		return nil, err
	}

	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
	i.elastic.BatchPersist()

	return nft, nil
}

func (i metadataIndexer) RefreshByStatus(status entity.MetadataStatus, metadataError string) error {
	size := 100
	page := 1

	for {
		nfts, total, err := i.nftRepo.GetMetadata(size, page, status, metadataError)
		if err != nil || len(nfts) == 0 {
			break
		}
		if page == 1 {
			zap.S().Infof("Processing %d %s NFTS", total, status)
		}

		for _, nft := range nfts {
			if metadataError != "" && metadataError != nft.Metadata.Error {
				continue
			}
			i.TriggerMetadataRefresh(nft)
		}
		i.elastic.BatchPersist()
		page++
	}
	i.elastic.Persist()

	return nil
}
