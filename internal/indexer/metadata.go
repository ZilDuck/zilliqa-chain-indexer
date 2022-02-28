package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)


type MetadataIndexer interface {
	TriggerMetadataRefresh(nft entity.Nft)
	TriggerAssetRefresh(nft entity.Nft)

	RefreshMetadata(contractAddr string, tokenId uint64) error
	RefreshAsset(contractAddr string, tokenId uint64, force bool) error
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
	return metadataIndexer{elastic, nftRepo, messageService, metadataService}
}

func (i metadataIndexer) TriggerMetadataRefresh(nft entity.Nft) {
	if nft.Metadata.UriEmpty() {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.MetadataRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue metadata refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger MetaData Refresh")
}

func (i metadataIndexer) TriggerAssetRefresh(nft entity.Nft) {
	if nft.Metadata.UriEmpty() {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.AssetRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue asset refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger Asset Refresh")
}

func (i metadataIndexer) RefreshMetadata(contractAddr string, tokenId uint64) error {
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId)).Info("NFT Refresh Metadata")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		return err
	}

	data, err := i.metadataService.FetchMetadata(*nft)
	if err != nil {
		zap.L().With(
			zap.Error(err),
			zap.String("contractAddr", nft.Contract),
			zap.Uint64("tokenId", nft.TokenId),
			zap.String("baseUrl", nft.BaseUri),
			zap.String("tokenUri", nft.TokenUri),
		).Warn("Failed to get zrc6 metadata")

		nft.Metadata.Error = err.Error()
		nft.Metadata.Attempted++
		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
		i.elastic.BatchPersist()

		return err
	}

	nft.Metadata.Data = data
	nft.Metadata.Error = ""

	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
	i.elastic.BatchPersist()

	return nil
}

func (i metadataIndexer) RefreshAsset(contractAddr string, tokenId uint64, force bool) error {
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId)).Info("NFT Refresh Asset")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().Error("Failed to find NFT for asset refresh")
		return err
	}

	err = i.metadataService.FetchImage(*nft, force)
	if err != nil {
		if errors.Is(err, metadata.ErrorAssetAlreadyExists) {
			zap.L().Warn("Asset already exists")
		} else {
			zap.L().With(zap.Error(err)).Error("Failed to fetch zrc6 asset")

			nft.Metadata.AssetError = err.Error()
			nft.Metadata.AssetAttempted++
			i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftAsset)
			i.elastic.BatchPersist()

			return err
		}
	}

	nft.MediaUri = fmt.Sprintf("%s/%d", contractAddr, tokenId)
	nft.Metadata.AssetError = ""

	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftAsset)
	i.elastic.BatchPersist()

	return nil
}

