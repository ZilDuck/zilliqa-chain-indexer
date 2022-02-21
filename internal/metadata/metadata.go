package metadata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/helper"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Service interface {
	FetchZrc6Metadata(nft entity.Nft) (map[string]interface{}, error)
	FetchZrc6Image(nft entity.Nft) (string, error)
}

type service struct {
	client    *retryablehttp.Client
	ipfsHosts []string
	assetPath string
}

type Metadata map[string]interface{}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0"

var (
	ErrorAssetAlreadyExists = errors.New("asset already exists")
)

func NewMetadataService(client *retryablehttp.Client, ipfsHosts []string, assetPath string) Service {
	return service{client, ipfsHosts, assetPath}
}

func (s service) FetchZrc6Metadata(nft entity.Nft) (map[string]interface{}, error) {
	if nft.Metadata == nil {
		return nil, errors.New("metadata uri not valid")
	}

	if nft.Metadata.Ipfs {
		resp, err := s.fetchIpfs(nft.Metadata.Uri[7:])
		if err != nil {
			return nil, err
		}

		md, err := s.hydrateMetadata(resp)
		if err != nil {
			return nil, err
		}

		return md, nil
	}

	return s.fetchHttp(nft)
}

func (s service) fetchIpfs(ipfsCode string) (*http.Response, error) {
	hosts := s.ipfsHosts
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(hosts), func(i, j int) { hosts[i], hosts[j] = hosts[j], hosts[i] })

	for _, host := range hosts {
		uri := fmt.Sprintf("%s/ipfs/%s", host, ipfsCode)
		zap.S().Debugf("Fetching IPFS metadata from %s", uri)
		req, err := retryablehttp.NewRequest("GET", uri, nil)
		if err != nil {
			continue
		}

		c1 := make(chan *http.Response, 1)

		go func() {
			resp, err := s.client.Do(req)
			if err != nil {
				c1 <- nil
			} else {
				c1 <- resp
			}
		}()

		select {
		case resp := <-c1:
			if resp == nil {
				continue
			}
			if resp.StatusCode != 200 {
				zap.S().With(zap.String("uri", uri)).Errorf("IPFS status code: %d", resp.StatusCode)
				continue
			}
			return resp, nil
		case <-time.After(5 * time.Second):
			zap.S().Infof("Timedout waiting for IPFS...next")
			continue
		}
	}

	return nil, errors.New("failed to fetch ipfs")
}

func (s service) fetchHttp(nft entity.Nft) (Metadata, error) {
	req, err := retryablehttp.NewRequest("GET", nft.Metadata.Uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	return s.hydrateMetadata(resp)
}

func (s service) hydrateMetadata(resp *http.Response) (Metadata, error) {
	if resp.StatusCode != 200 {
		zap.L().With(zap.String("status", resp.Status)).Error("Metadata fetch non 200 response")
		return nil, errors.New("metadata fetch non 200 response")
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	var md Metadata
	if err := json.Unmarshal(buf.Bytes(), &md); err != nil {
		return nil, err
	}

	return md, nil
}

func (s service) FetchZrc6Image(nft entity.Nft) (string, error) {
	if nft.Metadata == nil {
		return "", errors.New("metadata uri not valid")
	}

	assetUri, err := nft.Metadata.GetAssetUri()
	if err != nil {
		zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Error(err.Error())
	}

	contractDir := fmt.Sprintf("%s/%s", s.assetPath, nft.Contract)
	assetPath := fmt.Sprintf("%s/%d", contractDir, nft.TokenId)

	if _, err := os.Stat(assetPath); err == nil {
		return "", ErrorAssetAlreadyExists
	}

	if helper.IsIpfs(assetUri) {
		resp, err := s.fetchIpfs(helper.GetIpfs(assetUri)[7:])
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		_ = os.MkdirAll(contractDir, os.ModePerm)
		out, err := os.Create(assetPath)
		if err != nil {
			return "", err
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return "", err
		}
	}

	return assetUri, nil
}