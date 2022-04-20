package bunny

import (
	"encoding/json"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/messenger"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

type Service interface {
	TriggerPurge(el interface{})
	Purge(contractAddr string, tokenId uint64) error
}

type service struct {
	messageService messenger.MessageService
	client         *retryablehttp.Client
	apiKey         string
	cdnUrl         string
}

func NewBunnyService(messageService messenger.MessageService, client *retryablehttp.Client, apiKey, cdnUrl string) Service {
	return service{messageService, client, apiKey, cdnUrl}
}

func (s service) TriggerPurge(el interface{}) {
	if !config.Get().EventsSupported {
		return
	}

	nft := el.(entity.Nft)

	msgJson, _ := json.Marshal(messenger.Nft{Contract: nft.Contract, TokenId: nft.TokenId})
	if err := s.messageService.SendMessage(messenger.CdnPurge, msgJson, false); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to queue cdn purge")
	} else {
		zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Info("Trigger CDN Purge")
	}
}

func (s service) Purge(contractAddr string, tokenId uint64) error {
	url := fmt.Sprintf("https://api.bunny.net/purge?url=%s/%s/%d", s.cdnUrl, contractAddr, tokenId)
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to purge cache: %s", resp.Status)
	}

	return nil
}