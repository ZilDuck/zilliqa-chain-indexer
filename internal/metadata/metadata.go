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
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Service interface {
	FetchMetadata(nft entity.Nft) (map[string]interface{}, error)
	FetchImage(nft entity.Nft, force bool) error

	GetNftMedia(nft entity.Nft) ([]byte, string, error)
}

type service struct {
	client      *retryablehttp.Client
	ipfsHosts   []string
	assetPath   string
	ipfsTimeout int
}

type Metadata map[string]interface{}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0"

var (
	ErrorAssetAlreadyExists = errors.New("asset already exists")
)

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

func (s service) FetchImage(nft entity.Nft, force bool) error {
	if nft.Metadata.UriEmpty() {
		return errors.New("metadata uri not valid")
	}

	assetUri, err := nft.Metadata.GetAssetUri()
	if err != nil {
		zap.L().With(zap.String("contract", nft.Contract), zap.Uint64("tokenId", nft.TokenId)).Error(err.Error())
	}

	contractDir := fmt.Sprintf("%s/%s", s.assetPath, nft.Contract)
	zap.S().Debugf("Create asset folder for contract (if not exists): %s", contractDir)
	if err = os.MkdirAll(contractDir, os.ModePerm); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to create contract dir")
	}

	assetPath := fmt.Sprintf("%s/%d", contractDir, nft.TokenId)
	zap.S().Debugf("Using asset path: %s", assetPath)
	if _, err := os.Stat(assetPath); err == nil && !force {
		return ErrorAssetAlreadyExists
	}

	var resp *http.Response
	var respErr error

	if helper.IsIpfs(assetUri) {
		ipfsUri := helper.GetIpfs(assetUri)
		resp, respErr = s.fetchIpfs(*ipfsUri)
		if respErr != nil {
			zap.L().With(zap.Error(err)).Error("Failed to fetch image from ipfs")
			return respErr
		}
	} else {
		resp, respErr = s.fetchHttp(assetUri)
		if respErr != nil {
			zap.L().With(zap.Error(err)).Error("Failed to fetch image from http")
			return respErr
		}
	}

	defer resp.Body.Close()

	out, err := os.Create(assetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}

func (s service) fetchIpfs(uri string) (*http.Response, error) {
	hosts := s.ipfsHosts
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(hosts), func(i, j int) { hosts[i], hosts[j] = hosts[j], hosts[i] })

	for _, host := range hosts {
		uri := fmt.Sprintf("%s/ipfs/%s", host, uri[7:])
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
		case <-time.After(time.Duration(s.ipfsTimeout) * time.Second):
			zap.S().Warnf("Timedout waiting for IPFS...next")
			continue
		}
	}

	return nil, errors.New("failed to fetch ipfs")
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

func (s service) GetNftMedia(nft entity.Nft) ([]byte, string, error) {
	if s.assetPath == "" || nft.MediaUri == "" {
		return nil, "", errors.New("media not found")
	}

	buffer, err := ioutil.ReadFile(fmt.Sprintf("%s/%s",s.assetPath, nft.MediaUri))
	if err != nil {
		return nil, "", err
	}

	fileType, err := getFileContentType(buffer[:512])
	if err != nil {
		return nil, "", err
	}

	return buffer, fileType, nil
}

func getFileContentType(b []byte) (string, error) {
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	return http.DetectContentType(b), nil
}
