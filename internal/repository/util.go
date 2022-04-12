package repository

import (
	"context"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"strings"
	"time"
)

func search(searchService *elastic.SearchService) (*elastic.SearchResult, error) {
	result, err := searchService.Do(context.Background())
	if err != nil {
		if err.Error() == "elastic: Error 429 (Too Many Requests)" {
			zap.L().Warn("Elastic: 429 (Too Many Requests)")
			time.Sleep(5 * time.Second)
			return search(searchService)
		}
		if strings.Contains(err.Error(), "GOAWAY") {
			zap.L().Warn("Elastic: Transport received Server's graceful shutdown GOAWAY")
			time.Sleep(5 * time.Second)
			return search(searchService)
		}
		if strings.Contains(err.Error(), "no available connection") {
			zap.L().Warn("Elastic: no available connection: no Elasticsearch node available")
			time.Sleep(5 * time.Second)
			return search(searchService)
		}
	}

	return result, err
}
