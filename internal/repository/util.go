package repository

import (
	"context"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"time"
)

func search(searchService *elastic.SearchService) (*elastic.SearchResult, error) {
	result, err := searchService.Do(context.Background())
	if err != nil && err.Error() == "elastic: Error 429 (Too Many Requests)" {
		zap.L().Warn("Elastic: 429 (Too Many Requests)")
		time.Sleep(5 * time.Second)
		return search(searchService)
	}

	return result, err
}
