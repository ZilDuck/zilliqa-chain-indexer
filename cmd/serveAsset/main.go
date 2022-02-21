package main

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var container *dic.Container

func main() {
	config.Init()

	container, _ = dic.NewContainer()

	r := mux.NewRouter()
	r.HandleFunc("/{contractAddr}/{tokenId}", GetAsset).Methods("GET")

	zap.L().Info("Serving assets on :"+config.Get().AssetPort)
	err := http.ListenAndServe(":"+config.Get().AssetPort, r)
	if err != nil {
		panic(err)
	}
}

func GetAsset(w http.ResponseWriter, r *http.Request) {
	contractAddr, err := getContractAddr(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tokenId, err := getTokenId(r)
	if err != nil {
		http.Error(w, "Invalid tokenId", http.StatusBadRequest)
		return
	}

	nft, err := container.GetNftRepo().GetNft(contractAddr, tokenId)
	if err != nil {
		zap.L().With(zap.String("contractAddr", contractAddr), zap.Uint64("tokenId", tokenId), zap.Error(err)).Warn("NFT not found")
		http.Error(w, "NFT not found", http.StatusNotFound)
		return
	}

	if nft.MediaUri == "" {
		zap.L().With(zap.Error(err)).Warn("NFT Media URI not found")
		http.Error(w, "NFT not found", http.StatusNotFound)
		return
	}

	media, contentType, err := container.GetMetadataService().GetZrc6Media(*nft)
	if err != nil {
		zap.L().With(zap.Error(err)).Warn("Failed to get zrc6 media")
		http.Error(w, "Failed to get zrc6 media", http.StatusNotFound)
		return
	}

	w.WriteHeader(200)
	w.Header().Add("Content-Type", contentType)
	_, _ = fmt.Fprint(w, string(media[:]))
}

func getContractAddr(r *http.Request) (string, error) {
	contractAddr, ok := mux.Vars(r)["contractAddr"]
	if !ok {
		return "", errors.New("invalid parameters")
	}

	return contractAddr, nil
}

func getTokenId(r *http.Request) (uint64, error) {
	tokenId, ok := mux.Vars(r)["tokenId"]
	if !ok {
		return 0, errors.New("invalid parameters")
	}

	return strconv.ParseUint(tokenId, 10, 64)
}