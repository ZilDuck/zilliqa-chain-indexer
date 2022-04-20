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
	"net/http"
	"strings"
	"time"
)

type Service interface {
	FetchMetadata(nft entity.Nft) (map[string]interface{}, error)
	FetchImage(nft entity.Nft) (io.ReadCloser, error)
}

type service struct {
	client      *retryablehttp.Client
	ipfsHosts   []string
	ipfsTimeout int
}

type Metadata map[string]interface{}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0"

var (
	ErrNoSuchHost = errors.New("no such host")
	ErrNotFound = errors.New("404 not found")
	ErrBadRequest = errors.New("bad request")
	ErrInvalidContent = errors.New("invalid content")
	ErrUnsupportedProtocolScheme = errors.New("unsupported protocol scheme")
	ErrMetadataNotFound = errors.New("metadata not found")
	ErrTimeout = errors.New("timeout")
)

func NewMetadataService(client *retryablehttp.Client, ipfsHosts []string, ipfsTimeout int) Service {
	return service{client, ipfsHosts, ipfsTimeout}
}

func (s service) FetchMetadata(nft entity.Nft) (map[string]interface{}, error) {
	zap.L().With(zap.Uint64("tokenId", nft.TokenId), zap.String("contract", nft.Contract)).Info("Fetch metadata")
	if nft.Metadata.UriEmpty() {
		return nil, errors.New("metadata uri not valid")
	}

	var resp *http.Response
	var err error

	if nft.Metadata.IsIpfs {
		resp, err = s.fetchIpfs(nft.Metadata.Uri)
	} else {
		resp, err = s.fetchHttp(nft.Metadata.Uri)
	}

	if err != nil {
		if errors.Is(err, ErrTimeout) || errors.Is(err, ErrMetadataNotFound) || errors.Is(err, ErrNotFound) {
			return nil, err
		}
		if strings.Contains(err.Error(), "unsupported protocol scheme") {
			return nil, ErrUnsupportedProtocolScheme
		}
		if len(err.Error()) > 12 && err.Error()[len(err.Error())-12:] == "no such host" {
			return nil, ErrNoSuchHost
		}
		return nil, err
	}

	return s.hydrateMetadata(resp)
}

func (s service) FetchImage(nft entity.Nft) (io.ReadCloser, error) {
	assetUri := nft.AssetUri
	if assetUri == "" {
		if nft.Metadata.UriEmpty() {
			zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Warn("Metadata not found")
			return nil, errors.New("metadata uri not valid")
		}

		var err error
		assetUri, err = nft.Metadata.GetAssetUri()
		if err != nil {
			zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Error(err.Error())
		}
	}

	var resp *http.Response
	var respErr error

	if helper.IsIpfs(assetUri) {
		ipfsUri := helper.GetIpfs(assetUri, nil)
		resp, respErr = s.fetchIpfs(*ipfsUri)
		if respErr != nil {
			zap.L().With(zap.Error(respErr), zap.String("assetUri", assetUri)).Error("Failed to fetch image from ipfs")
			return nil, respErr
		}
	} else {
		resp, respErr = s.fetchHttp(assetUri)
		if respErr != nil {
			zap.L().With(zap.Error(respErr), zap.String("assetUri", assetUri)).Error("Failed to fetch image from http")
			return nil, respErr
		}
	}

	return resp.Body, nil
}

func (s service) fetchIpfs(uri string) (*http.Response, error) {
	ch := make(chan *http.Response, len(s.ipfsHosts))
	complete := 0

	for _, host := range s.ipfsHosts {
		go func(host string) {
			ipfsUri := fmt.Sprintf("%s/ipfs/%s", host, uri[7:])
			req, err := retryablehttp.NewRequest("GET", ipfsUri, nil)
			if err != nil {
				ch <- nil
				return
			}

			zap.L().With(zap.String("uri", uri), zap.String("ipfs", ipfsUri)).Info("Fetching IPFS metadata")
			resp, err := s.client.Do(req)
			if err != nil {
				ch <- nil
			}
			ch <- resp
		}(host)
	}

	for {
		select {
		case resp := <-ch:
			if resp != nil {
				if resp.StatusCode == 200 {
					return resp, nil
				}
				zap.S().With(zap.String("uri", uri)).Errorf("IPFS status code: %d", resp.StatusCode)
			}
			complete++
		case <-time.After(time.Duration(s.ipfsTimeout) * time.Second):
			zap.S().Warnf("Timedout waiting for IPFS...next")
			complete++
		}

		if complete == len(s.ipfsHosts) {
			break
		}
	}

	return nil, ErrMetadataNotFound
}

type fetchResponse struct {
	resp *http.Response
	err  error
}

func (s service) fetchHttp(uri string) (*http.Response, error) {
	ch := make(chan fetchResponse, 1)

	req, err := retryablehttp.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)
	zap.L().With(zap.String("uri", uri)).Debug("Fetching HTTP metadata")

	go func() {
		resp, err := s.client.Do(req)
		if err != nil {
			ch <-fetchResponse{err: err}
			return
		}

		if resp.StatusCode != 200 {
			if resp.StatusCode == http.StatusNotFound {
				ch <-fetchResponse{err: ErrNotFound}
				return
			}
			if resp.StatusCode == http.StatusBadRequest {
				ch <-fetchResponse{err: ErrBadRequest}
				return
			}
			zap.S().With(zap.String("uri", uri)).Errorf("HTTP status code: %d", resp.StatusCode)
			ch <-fetchResponse{err: errors.New(resp.Status)}
		}

		ch <- fetchResponse{resp: resp}
	}()
	select {
	case resp := <-ch:
		return resp.resp, resp.err
	case <-time.After(1 * time.Second):
		zap.L().Debug("Timed out waiting for "+uri)
		return nil, ErrTimeout
	}
}

func (s service) hydrateMetadata(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	var md map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &md); err != nil {
		return nil, ErrInvalidContent
	}

	return md, nil
}
