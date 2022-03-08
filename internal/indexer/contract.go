package indexer

import (
	"encoding/json"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/elastic_search"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/entity"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/factory"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/repository"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/zilliqa"
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"go.uber.org/zap"
	"sync"
)

type ContractIndexer interface {
	Index(txs []entity.Transaction) error
	BulkIndex(fromBlockNum uint64) error
	IndexContractState(contractAddr string, blockNun uint64, create bool) error
}

type contractIndexer struct {
	elastic      elastic_search.Index
	zilliqa      zilliqa.Service
	factory      factory.ContractFactory
	txRepo       repository.TransactionRepository
	contractRepo repository.ContractRepository
	nftRepo      repository.NftRepository
}

func NewContractIndexer(
	elastic elastic_search.Index,
	zilliqa zilliqa.Service,
	factory factory.ContractFactory,
	txRepo repository.TransactionRepository,
	contractRepo repository.ContractRepository,
	nftRepo repository.NftRepository,
) ContractIndexer {
	return contractIndexer{elastic, zilliqa, factory, txRepo, contractRepo, nftRepo}
}

func (i contractIndexer) Index(txs []entity.Transaction) error {
	for _, tx := range txs {
		if tx.Receipt.Success == false {
			continue
		}

		if tx.IsContractCreation {
			c, err := i.factory.CreateContractFromTx(tx)
			if err == nil {
				zap.L().With(
					zap.Uint64("blockNum", c.BlockNum),
					zap.String("name", c.Name),
					zap.String("address", c.Address),
				).Info("Index contract")

				i.elastic.AddIndexRequest(elastic_search.ContractIndex.Get(), c, elastic_search.ContractCreate)
			}

			_ = i.IndexContractState(c.Address, tx.BlockNum, true)
		}

		if tx.IsContractExecution {
			var wg sync.WaitGroup
			for _, contractAddr := range tx.GetEngagedContracts() {
				wg.Add(1)
				go func(addr string) {
					defer wg.Done()
					_ = i.IndexContractState(addr, tx.BlockNum, false)
				}(contractAddr)
			}
			wg.Wait()
		}
	}

	return nil
}

func (i contractIndexer) BulkIndex(fromBlockNum uint64) error {
	zap.L().With(zap.Uint64("from", fromBlockNum)).Info("Bulk index contracts")
	size := 100
	page := 1

	for {
		txs, _, err := i.txRepo.GetContractCreationTxs(fromBlockNum, size, page)
		if err != nil {
			zap.L().With(zap.Error(err)).Error("Failed to get contract txs")
			return err
		}
		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			c, err := i.factory.CreateContractFromTx(tx)
			if err != nil {
				continue
			}

			zap.L().With(
				zap.Uint64("blockNum", c.BlockNum),
				zap.String("name", c.Name),
				zap.String("address", c.Address),
			).Info("Index contract")

			i.elastic.AddIndexRequest(elastic_search.ContractIndex.Get(), *c, elastic_search.ContractCreate)

			_ = i.IndexContractState(c.Address, tx.BlockNum, true)

			i.elastic.BatchPersist()
		}

		i.elastic.Persist()

		page++
	}

	i.elastic.Persist()

	return nil
}

func (i contractIndexer) IndexContractState(contractAddr string, blockNum uint64, create bool) error {
	bech32Addr, _ := bech32.ToBech32Address(contractAddr)

	state, err := i.zilliqa.GetContractState(bech32Addr)
	if err != nil {
		return err
	}

	cState := entity.ContractState{
		Address:  contractAddr,
		BlockNum: blockNum,
		State:    make([]entity.ContractStateElement, 0),
	}

	for k, v := range state {
		switch v.(type) {
		case map[string]interface{}:
			vJson, _ := json.Marshal(v)
			cState.State = append(cState.State, entity.ContractStateElement{Key: k, Value: string(vJson)})
		case []interface{}:
			vJson, _ := json.Marshal(v)
			cState.State = append(cState.State, entity.ContractStateElement{Key: k, Value: string(vJson)})
		default:
			cState.State = append(cState.State, entity.ContractStateElement{Key: k, Value: v.(string)})
		}
	}

	zap.L().With(zap.String("address", contractAddr)).Info("Index contract state")

	if create {
		i.elastic.AddIndexRequest(elastic_search.ContractStateIndex.Get(), cState, elastic_search.ContractState)
	} else {
		i.elastic.AddUpdateRequest(elastic_search.ContractStateIndex.Get(), cState, elastic_search.ContractState)
	}

	return nil
}