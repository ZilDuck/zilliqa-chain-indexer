package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/event"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
	"math/rand"
	"time"
)

type MetadataIndexer interface {
	TriggerMetadataRefresh(el interface{})
	RefreshMetadata(contractAddr string, tokenId uint64) (*entity.Nft, error)
	RefreshByStatus(status entity.MetadataStatus) error
}

type metadataIndexer struct {
	elastic         elastic_search.Index
	nftRepo         repository.NftRepository
	contractRepo    repository.ContractRepository
	messageService  messenger.MessageService
	metadataService metadata.Service
}

func NewMetadataIndexer(
	elastic elastic_search.Index,
	nftRepo repository.NftRepository,
	contractRepo repository.ContractRepository,
	messageService messenger.MessageService,
	metadataService metadata.Service,
) MetadataIndexer {
	i := metadataIndexer{elastic, nftRepo, contractRepo, messageService, metadataService}

	event.AddEventListener(event.NftMintedEvent, i.TriggerMetadataRefresh)
	event.AddEventListener(event.ContractBaseUriUpdatedEvent, i.TriggerMetadataRefresh)
	event.AddEventListener(event.TokenUriUpdatedEvent, i.TriggerMetadataRefresh)

	return i
}

func (i metadataIndexer) TriggerMetadataRefresh(el interface{}) {
	if !config.Get().EventsSupported {
		return
	}

	nft := el.(entity.Nft)

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.MetadataRefresh, msgJson); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to queue metadata refresh")
	} else {
		zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger MetaData Refresh")
	}
}

func (i metadataIndexer) RefreshMetadata(contractAddr string, tokenId uint64) (*entity.Nft, error) {
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId)).Info("NFT Refresh Metadata")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		return nil, err
	}

	c, err := i.contractRepo.GetContractByAddress(contractAddr)
	if err != nil {
		return nil, err
	}

	properties, mimeType, err := i.metadataService.FetchMetadata(*nft)
	if err != nil {
		if err == metadata.ErrInvalidContent {
			if len(mimeType) > 5 && mimeType[:5] == "image" {
				nft.HasMetadata = false
				nft.Metadata = nil
				if helper.IsIpfs(nft.TokenUri) {
					nft.AssetUri = *helper.GetIpfs(nft.TokenUri, nil)
				} else {
					nft.AssetUri = nft.TokenUri
				}

				i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
				i.elastic.BatchPersist()

				return nft, nil
			}
		}

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
	nft.HasMetadata = true

	// ZilMorphs are fudge
	if nft.Contract == "0x852c4105660ab288d0df8b2491f7462c66a1c0ae" {
		nft.Metadata.Properties["image"] = fmt.Sprintf("https://zilmorphs.com/morph/%d.png", nft.TokenId)
	}

	if assetUri, err := nft.Metadata.GetAssetUri(); err == nil {
		if helper.IsIpfs(assetUri) {
			ipfsUri := helper.GetIpfs(assetUri, c)
			if ipfsUri == nil {
				zap.S().With(
					zap.String("assetUri", assetUri),
					zap.String("contract", contractAddr),
					zap.Uint64("tokenId", tokenId),
				).Error("IPFS not found")
				return nil, errors.New(fmt.Sprintf("Metadata is ipfs asset. Helper failed to retrieve (%s, %d)", contractAddr, tokenId))
			}
			assetUri = *ipfsUri
		}
		nft.AssetUri = assetUri
	} else {
		zap.L().With(
			zap.Error(err),
			zap.String("contract", contractAddr),
			zap.Uint64("tokenId", tokenId),
		).Error("Failed to get assetUri")
	}

	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
	i.elastic.BatchPersist()

	return nft, nil
}

func (i metadataIndexer) RefreshByStatus(status entity.MetadataStatus) error {
	size := 100
	page := 1

	for {
		nfts, total, err := i.nftRepo.GetMetadata(size, page, status)
		if err != nil || len(nfts) == 0 {
			break
		}
		if page == 1 {
			zap.S().Infof("Processing %d %s NFTS", total, status)
		}

		for _, nft := range nfts {
			if nft.Contract == "0x3fe64e8b3e9e110db331b32ea26e191c07f14f80" ||
				nft.Contract == "0x32e4df3cd46c30862b0a30cdb187045b11ee8753" ||
				nft.Contract == "0x821aea19180b0868f22301147f0c28204283d167" {
				continue
			}
			if nft.Metadata.Attempts == 0 || rand.Intn(nft.Metadata.Attempts+1) == 0 {
				i.TriggerMetadataRefresh(nft)
			}
		}
		i.elastic.BatchPersist()
		page++
	}
	i.elastic.Persist()

	return nil
}
