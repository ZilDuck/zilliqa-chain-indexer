package transaction

import (
	"errors"
	"github.com/patrickmn/go-cache"
)

type Service interface {
	SetLastBlockNumIndexed(blockNum uint64)
	GetLastBlockNumIndexed() (uint64, error)
	ClearLastBlockNumIndexed()
}

type service struct {
	repository Repository
	cache      *cache.Cache
}

func NewService(repository Repository, cache *cache.Cache) Service {
	return service{repository, cache}
}

var (
	ErrTxBlockTransactionNotFound = errors.New("transaction not found")
)

func (s service) ClearLastBlockNumIndexed() {
	s.cache.Delete("lastBlockNumIndexed")
}

func (s service) SetLastBlockNumIndexed(blockNum uint64) {
	s.cache.Set("lastBlockNumIndexed", blockNum, cache.NoExpiration)
}

func (s service) GetLastBlockNumIndexed() (uint64, error) {
	if lastBlockNumIndexed, exists := s.cache.Get("lastBlockNumIndexed"); exists {
		blockNum := lastBlockNumIndexed.(uint64)
		return blockNum, nil
	}

	blockNum, err := s.repository.GetBestBlockNum()
	if err != nil {
		return 0, err
	}
	s.SetLastBlockNumIndexed(blockNum)

	return blockNum, nil
}
