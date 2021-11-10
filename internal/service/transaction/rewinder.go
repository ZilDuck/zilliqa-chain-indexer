package transaction

import (
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"go.uber.org/zap"
)

type Rewinder interface {
	Rewind(blockNum uint64) error
}

type rewinder struct {
	elastic elastic_cache.Index
}

func NewRewinder(elastic elastic_cache.Index) Rewinder {
	return rewinder{elastic}
}

func (r rewinder) Rewind(blockNum uint64) error {
	zap.L().With(zap.Uint64("blockNum", blockNum)).Info("Rewinding transaction index")

	return r.elastic.DeleteBlockNumGT(blockNum, elastic_cache.TransactionIndex.Get())
}
