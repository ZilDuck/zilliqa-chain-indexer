package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"time"
)

var (
	ErrNftNotFound = errors.New("nft not found")
)

const (
	MaxRetries = 3
)

type NftRepository interface {
	Exists(contract string, tokenId uint64) bool
	GetNft(contract string, tokenId uint64) (*entity.Nft, error)
	GetNfts(contract string, size, page int) ([]entity.Nft, int64, error)
	GetBestTokenId(contractAddr string, blockNum uint64) (uint64, error)
	GetAllNfts(size, page int) ([]entity.Nft, int64, error)
	GetAllZrc1Nfts(size, page int) ([]entity.Nft, int64, error)
	GetAllZrc6Nfts(size, page int) ([]entity.Nft, int64, error)
	GetIpfsMetadata(size, page int) ([]entity.Nft, int64, error)
	ResetMetadata(nft entity.Nft) error
	GetBestBlockNum() (uint64, error)
	PurgeActions(contractAddr string) error
	PurgeContract(contractAddr string) error
}

type nftRepository struct {
	elastic elastic_search.Index
}

func NewNftRepository(elastic elastic_search.Index) NftRepository {
	return nftRepository{elastic}
}

func (r nftRepository) Exists(contract string, tokenId uint64) bool {
	_, err := r.getNft(contract, tokenId, -1)
	return err == nil
}

func (r nftRepository) GetNft(contract string, tokenId uint64) (*entity.Nft, error) {
	return r.getNft(contract, tokenId, 1)
}

func (r nftRepository) getNft(contract string, tokenId uint64, attempt int) (*entity.Nft, error) {
	pendingRequest := r.elastic.GetRequest(entity.CreateNftSlug(tokenId, contract))
	if pendingRequest != nil {
		pendingNft := pendingRequest.Entity.(entity.Nft)
		return &pendingNft, nil
	}

	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", contract),
		elastic.NewTermQuery("tokenId", tokenId),
	)

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Query(query).
		Size(1))

	nft, err := r.findOne(result, err)
	if err != nil {
		if attempt == -1 || attempt == MaxRetries {
			return nft, err
		}
		zap.S().With(zap.String("contractAddr", contract), zap.Uint64("tokenId", tokenId)).Warnf("Failed to find NFT in repo. retry(%d)", attempt)
		time.Sleep(time.Second * 1)
		return r.getNft(contract, tokenId, attempt+1)
	}

	return nft, err
}

func (r nftRepository) GetNfts(contract string, size, page int) ([]entity.Nft, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", contract),
	)

	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Query(query).
		Size(size).
		Sort("tokenId", true).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r nftRepository) GetIpfsMetadata(size, page int) ([]entity.Nft, int64, error) {
	query := elastic.NewNestedQuery("metadata", elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("metadata.ipfs", true),
	))

	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Query(query).
		Size(size).
		Sort("tokenId", true).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r nftRepository) GetAllNfts(size, page int) ([]entity.Nft, int64, error) {
	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Size(size).
		Sort("blockNum", false).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r nftRepository) GetAllZrc1Nfts(size, page int) ([]entity.Nft, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("zrc1", true),
	)

	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Query(query).
		Size(size).
		Sort("tokenId", true).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r nftRepository) GetAllZrc6Nfts(size, page int) ([]entity.Nft, int64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("zrc6", true),
	)

	from := size*page - size

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Query(query).
		Size(size).
		Sort("tokenId", true).
		From(from).
		TrackTotalHits(true))

	return r.findMany(result, err)
}

func (r nftRepository) GetBestTokenId(contractAddr string, blockNum uint64) (uint64, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", contractAddr),
		elastic.NewRangeQuery("blockNum").Lt(blockNum),
	)

	result, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Query(query).
		Sort("tokenId", false).
		Size(1))

	nft, err := r.findOne(result, err)
	if err != nil {
		if errors.Is(ErrNftNotFound, err) {
			return 0, nil
		}
		return 0, err
	}

	return nft.TokenId, nil
}

func (r nftRepository) ResetMetadata(nft entity.Nft) error {
	query := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("contract.keyword", nft.Contract),
		elastic.NewTermQuery("tokenId", nft.TokenId),
	)

	_, err := r.elastic.GetClient().
		UpdateByQuery(elastic_search.NftIndex.Get()).
		Query(query).
		Script(elastic.NewScript("ctx._source.metadata.remove(\"data\")")).
		Do(context.Background())

	return err
}

func (r nftRepository) GetBestBlockNum() (uint64, error) {
	results, err := search(r.elastic.GetClient().
		Search(elastic_search.NftIndex.Get()).
		Size(1).
		Sort("blockNum", false))

	nft, err := r.findOne(results, err)
	if err != nil {
		return 0, err
	}

	return nft.BlockNum, nil
}

func (r nftRepository) PurgeActions(contractAddr string) error {
	zap.L().With(zap.String("contractAddr", contractAddr)).Info("Purge actions for contract")

	_, err := r.elastic.GetClient().
		DeleteByQuery(elastic_search.NftActionIndex.Get()).
		Query(elastic.NewTermsQuery("contract.keyword", contractAddr)).
		Do(context.Background())

	if err != nil {
		zap.L().With(zap.Error(err), zap.String("contractAddr", contractAddr)).Error("Failed to purge nft actions")
	}

	return err
}

func (r nftRepository) PurgeContract(contractAddr string) error {
	zap.L().With(zap.String("contractAddr", contractAddr)).Info("Purge contract")

	_, err := r.elastic.GetClient().
		DeleteByQuery(elastic_search.NftIndex.Get()).
		Query(elastic.NewTermsQuery("contract.keyword", contractAddr)).
		Do(context.Background())

	if err != nil {
		zap.L().With(zap.Error(err), zap.String("contractAddr", contractAddr)).Error("Failed to purge contract")
	}

	return err
}

func (r nftRepository) findOne(results *elastic.SearchResult, err error) (*entity.Nft, error) {
	if err != nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, ErrNftNotFound
	}

	var nft entity.Nft
	hit := results.Hits.Hits[0]
	err = json.Unmarshal(hit.Source, &nft)

	return &nft, err
}

func (r nftRepository) findMany(results *elastic.SearchResult, err error) ([]entity.Nft, int64, error) {
	nfts := make([]entity.Nft, 0)

	if err != nil {
		return nfts, 0, err
	}

	for _, hit := range results.Hits.Hits {
		var nft entity.Nft
		if err := json.Unmarshal(hit.Source, &nft); err == nil {
			nfts = append(nfts, nft)
		}
	}

	return nfts, results.TotalHits(), nil
}
