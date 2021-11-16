package main

import (
	"github.com/dantudor/zil-indexer/generated/dic"
	"github.com/dantudor/zil-indexer/internal/config"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"time"
)

var container *dic.Container

func main() {
	initialize()
	defer sentry.Flush(2 * time.Second)

	container.GetDaemon().Execute()
}

func initialize() {
	config.Init()
	container, _ = dic.NewContainer()
	zap.L().Info("Indexer Started")
}
