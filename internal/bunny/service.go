package bunny

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/event"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"net/url"
)

type Service interface {
	PurgeCacheFromEvent(el interface{})
	PurgeCache(contractAddr string, tokenId uint64) error
}

type service struct {
	cdnUrl    string
	accessKey string
	client    *retryablehttp.Client
}

func NewService(cdnUrl, accessKey string, client *retryablehttp.Client) Service {
	s := service{cdnUrl, accessKey, client}
	event.AddEventListener(event.MetadataRefreshedEvent, s.PurgeCacheFromEvent)

	return s
}

func (s service) PurgeCacheFromEvent(el interface{}) {
	if !config.Get().EventsSupported {
		return
	}

	nft := el.(entity.Nft)

	s.PurgeCache(nft.Contract, nft.TokenId)
}

func (s service) PurgeCache(contractAddr string, tokenId uint64) error {
	zap.L().With(
		zap.String("contract", contractAddr),
		zap.Uint64("tokenId", tokenId),
	).Info("Bunny cache purge request")

	assetPath := url.QueryEscape(fmt.Sprintf("%s/%s/%d", s.cdnUrl, contractAddr, tokenId))
	uri := fmt.Sprintf("https://api.bunny.net/purge?url=%s", assetPath)

	req, err := retryablehttp.NewRequest("GET", uri, nil)
	if err != nil {
		zap.L().With(
			zap.Error(err),
			zap.String("uri", uri),
			zap.String("contract", contractAddr),
			zap.Uint64("tokenId", tokenId),
		).Error("Failed to create purge request")
		return err
	}
	req.Header.Set("AccessKey", s.accessKey)

	resp, err := s.client.Do(req)
	if err != nil {
		zap.L().With(
			zap.Error(err),
			zap.String("uri", uri),
			zap.String("contract", contractAddr),
			zap.Uint64("tokenId", tokenId),
		).Error("Failed to handle purge request")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		zap.L().With(
			zap.Int("status", resp.StatusCode),
			zap.String("uri", uri),
			zap.String("contract", contractAddr),
			zap.Uint64("tokenId", tokenId),
		).Error("Failed to handle purge request")
		return errors.New("bad status code")
	}

	zap.L().With(
		zap.String("contract", contractAddr),
		zap.Uint64("tokenId", tokenId),
	).Info("Bunny cache purge success")

	return nil
}
