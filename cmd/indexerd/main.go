package main

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/generated/dic"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

var container *dic.Container

func main() {
	config.Init("indexer")
	container, _ = dic.NewContainer()

	go health()

	zap.L().With(zap.String("port", config.Get().HealthPort)).Info("Indexer Started")

	container.GetDaemon().Execute()
}

func health() {
	if err := http.ListenAndServe(":"+config.Get().HealthPort, router()); err != nil {
		zap.L().With(zap.Error(err)).Error("Failed to start indexer")
	}
}

func router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "OK")
	}).Methods("GET")

	return r
}
