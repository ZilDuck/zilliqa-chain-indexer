package metadata

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/hashicorp/go-retryablehttp"
)

type Service interface {
	FetchZrc6Metadata(nft entity.Nft) (map[string]interface{}, error)
}

type service struct {
	client *retryablehttp.Client
}

func NewMetadataService(client *retryablehttp.Client) Service {
	return service{client}
}

func (s service) FetchZrc6Metadata(nft entity.Nft) (map[string]interface{}, error) {
	if nft.Metadata == nil {
		return nil, errors.New("metadata uri not valid")
	}

	req, err := retryablehttp.NewRequest("GET", nft.Metadata.Uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	var md map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &md); err != nil {
		return nil, err
	}

	return md, nil
}
