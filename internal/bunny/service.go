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
	cdnUrl     string
	cdnTestUrl string
	accessKey  string
	client     *retryablehttp.Client
}

func NewService(cdnUrl, cdnTestUrl, accessKey string, client *retryablehttp.Client) Service {
	s := service{cdnUrl, cdnTestUrl, accessKey, client}
	event.AddEventListener(event.MetadataRefreshedEvent, s.PurgeCacheFromEvent)

	return s
}

func (s service) PurgeCacheFromEvent(el interface{}) {
	if !config.Get().EventsSupported {
		zap.L().Warn("PurgeCacheFromEvent: Events disabled")
		return
	}
	zap.L().Info("PurgeCacheFromEvent")

	nft := el.(entity.Nft)

	_ = s.PurgeCache(nft.Contract, nft.TokenId)
}

func (s service) PurgeCache(contractAddr string, tokenId uint64) error {
	zap.L().With(
		zap.String("contract", contractAddr),
		zap.Uint64("tokenId", tokenId),
	).Info("Bunny cache purge request")

	args := []string{
		"",
		"?optimizer=image&width=800",
		"?&optimizer=image&height=400&width=400&aspect_ratio=1:1",
		"?optimizer=image&width=650",
		"?&optimizer=image&width=650",
	}

	urls := []string{s.cdnUrl, s.cdnTestUrl}
	for _, host := range urls {
		for _, arg := range args {
			assetPath := fmt.Sprintf("%s/%s/%d%s", host, contractAddr, tokenId, arg)
			zap.S().Debugf("Bunny purge: %s", assetPath)

			uri := fmt.Sprintf("https://api.bunny.net/purge?url=%s", url.QueryEscape(assetPath))
			req, _ := retryablehttp.NewRequest("GET", uri+arg, nil)
			req.Header.Set("AccessKey", s.accessKey)

			resp, err := s.client.Do(req)
			if err != nil {
				zap.L().With(
					zap.Error(err),
					zap.String("uri", uri+arg),
					zap.String("contract", contractAddr),
					zap.Uint64("tokenId", tokenId),
				).Error("Failed to handle purge request")
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				zap.L().With(
					zap.Int("status", resp.StatusCode),
					zap.String("uri", uri+arg),
					zap.String("contract", contractAddr),
					zap.Uint64("tokenId", tokenId),
				).Error("Failed to handle purge request")
				return errors.New("bad status code")
			}
		}
	}

	zap.L().With(
		zap.String("contract", contractAddr),
		zap.Uint64("tokenId", tokenId),
	).Info("Bunny cache purge success")

	return nil
}
