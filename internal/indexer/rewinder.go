package indexer

import (
	"github.com/dantudor/zil-indexer/internal/elastic_cache"
	"github.com/dantudor/zil-indexer/internal/service/contract"
	"github.com/dantudor/zil-indexer/internal/service/nft"
	"github.com/dantudor/zil-indexer/internal/service/transaction"
	"go.uber.org/zap"
)

type Rewinder interface {
	RewindToHeight(height uint64) error
}

type rewinder struct {
	elastic          elastic_cache.Index
	txRewinder       transaction.Rewinder
	txService        transaction.Service
	txRepo           transaction.Repository
	contractRewinder contract.Rewinder
	nftRewinder      nft.Rewinder
}

func NewRewinder(
	elastic elastic_cache.Index,
	txRewinder transaction.Rewinder,
	txService transaction.Service,
	txRepo transaction.Repository,
	contractRewinder contract.Rewinder,
	nftRewinder nft.Rewinder,
) Rewinder {
	return rewinder{
		elastic,
		txRewinder,
		txService,
		txRepo,
		contractRewinder,
		nftRewinder,
	}
}

func (r rewinder) RewindToHeight(height uint64) error {
	zap.L().With(zap.Uint64("height", height)).Info("Rewinding to height")

	r.elastic.ClearRequests()

	if err := r.txRewinder.Rewind(height); err != nil {
		zap.L().With(zap.Error(err)).Fatal("Failed to rewind")
		return err
	}

	zap.L().With(zap.Uint64("height", height)).Info("Rewound to height")
	r.elastic.Persist()

	return nil
}
