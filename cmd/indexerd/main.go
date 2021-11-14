package main

import (
	"github.com/dantudor/zil-indexer/internal/daemon"
	"github.com/getsentry/sentry-go"
	"time"
)

func main() {
	defer sentry.Flush(2 * time.Second)

	daemon.Execute()
}
