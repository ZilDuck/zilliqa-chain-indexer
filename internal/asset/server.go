package asset

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/metadata"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Server struct {
	nftRepo         repository.NftRepository
	metadataService metadata.Service
}

func NewServer(nftRepo repository.NftRepository, metadataService metadata.Service) Server {
	return Server{nftRepo, metadataService}
}

func (s Server) Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", s.handleHomepage).Methods("GET")
	r.HandleFunc("/{contractAddr}/{tokenId}", s.handleGetAsset).Methods("GET")
	r.NotFoundHandler = notFoundHandler()

	return r
}

func (s Server) handleHomepage(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "Zilliqa Asset CDN")
}

func (s Server) handleGetAsset(w http.ResponseWriter, r *http.Request) {
	contractAddr, _ := mux.Vars(r)["contractAddr"]
	tokenId, _ := getTokenId(r)

	nft, err := s.nftRepo.GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().With(zap.Error(err)).Warn("NFT not available")

		data, err := ioutil.ReadFile("static/missing-asset.png")
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Preview image not available")
			http.Error(w, "NFT not available", http.StatusNotFound)
		}
		w.WriteHeader(404)
		w.Header().Set("Content-Type", "image/png")
		_, _ = fmt.Fprint(w, string(data[:]))
		//http.Error(w, "NFT not available", http.StatusNotFound)
		return
	}

	body, err := s.metadataService.FetchImage(*nft)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("uri", nft.AssetUri)).Warn("NFT asset not available")
		http.Error(w, "NFT asset not available", http.StatusNotFound)
		return
	}
	defer body.Close()

	buf := new(bytes.Buffer)
	_, errBuf := buf.ReadFrom(body)
	if errBuf != nil {
		zap.L().With(zap.Error(errBuf)).Warn("Failed to process asset")
		http.Error(w, "Failed to process asset", http.StatusInternalServerError)
		return
	}

	data := buf.Bytes()

	contentType, err := getFileContentType(data[:512])

	w.WriteHeader(200)
	w.Header().Add("Content-Type", contentType)
	w.Header().Add("Cache-Control", "max-age: 31536000, immutable")
	_, _ = fmt.Fprint(w, string(data[:]))
	zap.L().With(zap.String("contract", contractAddr), zap.Uint64("tokenId", tokenId), zap.String("uri", nft.AssetUri)).Info("Serving nft")
}

func getTokenId(r *http.Request) (uint64, error) {
	tokenId, ok := mux.Vars(r)["tokenId"]
	if !ok {
		return 0, errors.New("invalid parameters")
	}

	return strconv.ParseUint(tokenId, 10, 64)
}

func getFileContentType(b []byte) (string, error) {
	return http.DetectContentType(b), nil
}

func notFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprintf(w, "Page not found")
	})
}
