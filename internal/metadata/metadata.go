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
	"net/http"
	"strings"
	"time"
)

type Service interface {
	FetchMetadata(nft entity.Nft) (map[string]interface{}, string, error)
	FetchImage(nft entity.Nft) (io.ReadCloser, error)

	FetchImageForContractMetadata(md entity.ContractMetadata) (io.ReadCloser, error)
	FetchContractMetadata(contract entity.Contract) (entity.ContractMetadata, error)
}

type service struct {
	client      *retryablehttp.Client
	ipfsHosts   []string
	ipfsTimeout int
}

type Metadata map[string]interface{}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0"

var (
	ErrNoSuchHost                = errors.New("no such host")
	ErrNotFound                  = errors.New("404 not found")
	ErrBadRequest                = errors.New("bad request")
	ErrInvalidContent            = errors.New("invalid content")
	ErrUnsupportedProtocolScheme = errors.New("unsupported protocol scheme")
	ErrMetadataNotFound          = errors.New("metadata not found")
	ErrTimeout                   = errors.New("timeout")
)

func NewMetadataService(client *retryablehttp.Client, ipfsHosts []string, ipfsTimeout int) Service {
	return service{client, ipfsHosts, ipfsTimeout}
}

func (s service) FetchMetadata(nft entity.Nft) (map[string]interface{}, string, error) {
	zap.L().With(zap.Uint64("tokenId", nft.TokenId), zap.String("contract", nft.Contract)).Info("Fetch metadata")
	if nft.Metadata.UriEmpty() {
		return nil, "", errors.New("metadata uri not valid")
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
			return nil, "", err
		}
		if strings.Contains(err.Error(), "unsupported protocol scheme") {
			return nil, "", ErrUnsupportedProtocolScheme
		}
		if len(err.Error()) > 12 && err.Error()[len(err.Error())-12:] == "no such host" {
			return nil, "", ErrNoSuchHost
		}

		return nil, "", err
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

func (s service) FetchImageForContractMetadata(md entity.ContractMetadata) (io.ReadCloser, error) {
	assetUri, ok := md["collection_image_url"]
	if !ok {
		zap.L().With(zap.String("contract", md["contract"].(string))).Warn("Contract metadata not found")
		return nil, errors.New("metadata uri not valid")
	}

	var resp *http.Response
	var respErr error

	if helper.IsIpfs(assetUri.(string)) {
		ipfsUri := helper.GetIpfs(assetUri.(string), nil)
		resp, respErr = s.fetchIpfs(*ipfsUri)
		if respErr != nil {
			zap.L().With(zap.Error(respErr), zap.String("assetUri", assetUri.(string))).Error("Failed to fetch image from ipfs")
			return nil, respErr
		}
	} else {
		resp, respErr = s.fetchHttp(assetUri.(string))
		if respErr != nil {
			zap.L().With(zap.Error(respErr), zap.String("assetUri", assetUri.(string))).Error("Failed to fetch image from http")
			return nil, respErr
		}
	}

	return resp.Body, nil
}

func (s service) FetchContractMetadata(contract entity.Contract) (entity.ContractMetadata, error) {
	var resp *http.Response
	var err error

	uri := fmt.Sprintf("%smetadata.json", contract.BaseUri)
	zap.L().Info(uri)
	if helper.IsIpfs(contract.BaseUri) {
		ipfsUri := helper.GetIpfs(uri, nil)
		zap.L().Info(*ipfsUri)
		resp, err = s.fetchIpfs(*ipfsUri)
	} else {
		resp, err = s.fetchHttp(uri)
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

	metadata, _, err := s.hydrateMetadata(resp)
	metadata["contract"] = contract.Address

	return metadata, err

}

type IpfsResp struct {
	uri  string
	resp *http.Response
	err  error
}

type IpfsCanceller struct {
	uri        string
	cancelFunc context.CancelFunc
}

func (s service) fetchIpfs(uri string) (*http.Response, error) {
	ch := make(chan IpfsResp, len(s.ipfsHosts))
	cancelCh := make(chan IpfsCanceller, len(s.ipfsHosts))
	complete := 0

	for _, host := range s.ipfsHosts {
		go func(host string) {
			ipfsUri := fmt.Sprintf("%s/ipfs/%s", host, uri[7:])

			ipfsResp := IpfsResp{uri: ipfsUri}

			ctx, cancel := context.WithCancel(context.Background())
			cancelCh <- IpfsCanceller{uri: ipfsUri, cancelFunc: cancel}

			req, err := retryablehttp.NewRequestWithContext(ctx, "GET", ipfsResp.uri, nil)
			if err != nil {
				ipfsResp.err = err
				ch <- ipfsResp
				return
			}

			zap.L().With(zap.String("ipfs", ipfsResp.uri)).Debug("Fetching IPFS metadata")
			resp, err := s.client.Do(req)
			if err != nil {
				ipfsResp.err = err
				zap.L().With(zap.String("ipfs", ipfsResp.uri), zap.Error(err)).Debug("Failed fetching metadata")
				ch <- ipfsResp
			}
			ipfsResp.resp = resp
			ch <- ipfsResp
		}(host)
	}

	cancelChannels := func(uri string) {
		cancelled := 0
		for {
			select {
			case resp := <-cancelCh:
				cancelled++
				if uri != resp.uri {
					zap.L().With(zap.String("ipfs", resp.uri)).Debug("Cancelling ipfs request")
					resp.cancelFunc()
				}
			}

			if cancelled == len(s.ipfsHosts)*s.client.RetryMax {
				break
			}
		}
	}

	for {
		select {
		case resp := <-ch:
			if resp.resp != nil {
				if resp.resp.StatusCode == 200 {
					zap.L().With(zap.String("ipfs", resp.uri)).Info("Complete ipfs request")
					cancelChannels(resp.uri)
					return resp.resp, nil
				}
			}
			complete++
		case <-time.After(time.Duration(s.ipfsTimeout) * time.Second):
			cancelChannels("")
			zap.S().With(zap.String("uri", uri)).Warnf("Timedout waiting for IPFS...next")
			return nil, ErrMetadataNotFound
		}

		if complete == len(s.ipfsHosts)*s.client.RetryMax {
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
			ch <- fetchResponse{err: err}
			return
		}

		if resp.StatusCode != 200 {
			if resp.StatusCode == http.StatusNotFound {
				ch <- fetchResponse{err: ErrNotFound}
				return
			}
			if resp.StatusCode == http.StatusBadRequest {
				ch <- fetchResponse{err: ErrBadRequest}
				return
			}
			zap.S().With(zap.String("uri", uri)).Errorf("HTTP status code: %d", resp.StatusCode)
			ch <- fetchResponse{err: errors.New(resp.Status)}
		}

		ch <- fetchResponse{resp: resp}
	}()
	select {
	case resp := <-ch:
		return resp.resp, resp.err
	case <-time.After(1 * time.Second):
		zap.L().Debug("Timed out waiting for " + uri)
		return nil, ErrTimeout
	}
}

func (s service) hydrateMetadata(resp *http.Response) (map[string]interface{}, string, error) {
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, "", err
	}

	var md map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &md); err != nil {
		return nil, http.DetectContentType(buf.Bytes()), ErrInvalidContent
	}

	return md, http.DetectContentType(buf.Bytes()), nil
}
