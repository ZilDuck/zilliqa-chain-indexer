package metadata

import (
	"bytes"
	"context"
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
	"sync"
	"time"
)

type Service interface {
	FetchMetadata(nft entity.Nft) (map[string]interface{}, error)
	FetchImage(nft entity.Nft) (io.ReadCloser, error)
}

type service struct {
	client      *retryablehttp.Client
	ipfsHosts   []string
	assetPath   string
	ipfsTimeout int
}

type Metadata map[string]interface{}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0"

func NewMetadataService(client *retryablehttp.Client, ipfsHosts []string, assetPath string, ipfsTimeout int) Service {
	return service{client, ipfsHosts, assetPath, ipfsTimeout}
}

func (s service) FetchMetadata(nft entity.Nft) (map[string]interface{}, error) {
	if nft.Metadata.UriEmpty() {
		return nil, errors.New("metadata uri not valid")
	}

	var resp *http.Response
	var err error

	if nft.Metadata.Ipfs {
		resp, err = s.fetchIpfs(nft.Metadata.Uri)
	} else {
		resp, err = s.fetchHttp(nft.Metadata.Uri)
	}

	if err != nil {
		return nil, err
	}

	return s.hydrateMetadata(resp)
}

func (s service) FetchImage(nft entity.Nft) (io.ReadCloser, error) {
	if nft.Metadata.UriEmpty() {
		return nil, errors.New("metadata uri not valid")
	}

	assetUri, err := nft.Metadata.GetAssetUri()
	if err != nil {
		zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Error(err.Error())
	}

	var resp *http.Response
	var respErr error

	if helper.IsIpfs(assetUri) {
		ipfsUri := helper.GetIpfs(assetUri)
		resp, respErr = s.fetchIpfs(*ipfsUri)
		zap.L().Info("Response received")
		if respErr != nil {
			zap.L().With(zap.Error(err), zap.String("assetUri", assetUri)).Error("Failed to fetch image from ipfs")
			return nil, respErr
		}
	} else {
		resp, respErr = s.fetchHttp(assetUri)
		if respErr != nil {
			zap.L().With(zap.Error(err), zap.String("assetUri", assetUri)).Error("Failed to fetch image from http")
			return nil, respErr
		}
	}

	return resp.Body, nil
}

func (s service) fetchIpfs(uri string) (*http.Response, error) {
	hosts := s.ipfsHosts
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(hosts), func(i, j int) { hosts[i], hosts[j] = hosts[j], hosts[i] })

	var wg sync.WaitGroup
	ch := make(chan *http.Response, len(hosts))

	ctx := context.Background()
	//ctx, cancel := context.WithCancel(ctx)

	for _, host := range hosts {
		wg.Add(1)
		host := host
		uri := fmt.Sprintf("%s/ipfs/%s", host, uri[7:])
		go func() {
			defer wg.Done()
			zap.L().With(zap.String("uri", uri)).Warn("Attempting to find IPFS asset")
			req, err := retryablehttp.NewRequest(http.MethodGet, uri, nil)
			if err != nil {
				zap.L().Error(err.Error())
				return
			}
			req = req.WithContext(ctx)

			resp, err := s.client.Do(req)
			if err != nil {
				zap.L().Error(err.Error())
				return
			}
			if resp != nil && resp.StatusCode == 200 {
				zap.L().With(zap.String("uri", uri)).Info("Response received")
				ch <- resp // no need to test the context, ch has rooms for this push to happen anyways.
			}
		}()
	}

	time.Sleep(time.Second)
	go func() {
		wg.Wait()
		close(ch)
	}()

	resp := <-ch
	winnerResp := resp

	//go func() {
	//	time.Sleep(time.Second)
	//	//cancel()
	//}()

	zap.L().With(zap.Int("statusCode", winnerResp.StatusCode)).Info("Responding")
	return winnerResp, nil
}

func (s service) fetchHttp(uri string) (*http.Response, error) {
	req, err := retryablehttp.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		zap.S().With(zap.String("uri", uri)).Errorf("HTTP status code: %d", resp.StatusCode)
		return nil, errors.New(resp.Status)
	}

	return resp, nil
}

func (s service) hydrateMetadata(resp *http.Response) (Metadata, error) {
	defer resp.Body.Close()

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
