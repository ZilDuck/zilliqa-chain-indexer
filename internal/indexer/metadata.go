package indexer

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/event"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"go.uber.org/zap"
)


type MetadataIndexer interface {
	TriggerMetadataRefresh(el interface{})
	TriggerAssetRefresh(el interface{})

	RefreshMetadata(contractAddr string, tokenId uint64) (*entity.Nft, error)
	//RefreshAsset(contractAddr string, tokenId uint64, force bool) error
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

	return i
}

func (i metadataIndexer) TriggerMetadataRefresh(el interface{}) {
	nft := el.(entity.Nft)

	if nft.Metadata.UriEmpty() {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.MetadataRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue metadata refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger MetaData Refresh")
}

func (i metadataIndexer) TriggerAssetRefresh(el interface{}) {
	nft := el.(entity.Nft)

	if nft.Metadata.UriEmpty() {
		return
	}

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := i.messageService.SendMessage(messenger.AssetRefresh, msgJson); err != nil {
		zap.L().Error("Failed to queue asset refresh")
	}
	zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger Asset Refresh")
}

func (i metadataIndexer) RefreshMetadata(contractAddr string, tokenId uint64) (*entity.Nft, error) {
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId)).Info("NFT Refresh Metadata")

	nft, err := i.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		return nil, err
	}

	data, err := i.metadataService.FetchMetadata(*nft)
	if err != nil {
		zap.L().With(
			zap.Error(err),
			zap.String("contractAddr", nft.Contract),
			zap.Uint64("tokenId", nft.TokenId),
			zap.String("baseUrl", nft.BaseUri),
			zap.String("tokenUri", nft.TokenUri),
		).Warn("Failed to get NFT metadata")

		nft.Metadata.Error = err.Error()
		nft.Metadata.Attempted++
		i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
		i.elastic.BatchPersist()

		return nil, err
	}

	nft.Metadata.Data = data
	nft.Metadata.Error = ""

	if err := i.nftRepo.ResetMetadata(*nft); err != nil {
		return nil, err
	}

	i.elastic.AddUpdateRequest(elastic_search.NftIndex.Get(), *nft, elastic_search.NftMetadata)
	i.elastic.BatchPersist()

	return nft, nil
}
