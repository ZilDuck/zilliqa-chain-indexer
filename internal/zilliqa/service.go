package zilliqa

import (
	"encoding/json"
	"fmt"
	"github.com/Zilliqa/gozilliqa-sdk/core"
	"go.uber.org/zap"
)

type Service interface {
	GetBlockchainInfo() (interface{}, error)
	GetDSBlock(height uint64) (interface{}, error)
	GetTxBlock(height uint64) (*core.TxBlock, error)
	GetTxBlocks(from uint64, count uint) ([]core.TxBlock, error)
	GetLatestTxBlock() (*core.TxBlock, error)
	GetTransactionsForTxBlock(height uint64) ([][]string, error)
	GetTxnBodiesForTxBlock(height uint64) ([]core.Transaction, error)
	GetTxnBodiesForTxBlocks(from, count uint64) (map[string][]core.Transaction, error)
	GetTransaction(hash string) (*core.Transaction, error)

	GetContractAddressFromTransactionID(txId string) (string, error)
	GetContractAddressFromTransactionIDs(txIds []string) (map[string]string, error)
	GetSmartContractInit(contractAddress string) ([]core.ContractValue, error)
	GetSmartContractInits(contractAddresses []string) ([][]core.ContractValue, error)
	GetSmartContractCode(contractAddress string) (string, error)
	GetContractState(contractAddress string) (map[string]interface{}, error)
	GetContractSubState(contractAddress string, params ...interface{}) (string, error)
}

type service struct {
	provider *Provider
}

func NewZilliqaService(provider *Provider) Service {
	return service{provider}
}

func (s service) GetBlockchainInfo() (interface{}, error) {
	blockchainInfo, err := s.provider.GetBlockchainInfo()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(blockchainInfo)
	zap.S().With(zap.String("data", string(b))).Infof("Blockchain Info")

	return nil, nil
}

func (s service) GetDSBlock(height uint64) (interface{}, error) {
	dsBlock, err := s.provider.GetDsBlock(fmt.Sprintf("%d", height))
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(dsBlock)
	zap.S().With(zap.String("data", string(b))).Infof("DS Block: %d", height)

	return nil, nil
}

func (s service) GetTxBlock(height uint64) (*core.TxBlock, error) {
	return s.provider.GetTxBlock(fmt.Sprintf("%d", height))
}

func (s service) GetTxBlocks(from uint64, count uint) ([]core.TxBlock, error) {
	blockNums := make([]string, 0)

	for x := from; x < from+uint64(count); x++ {
		blockNums = append(blockNums, fmt.Sprintf("%d", x))
	}

	return s.provider.GetTxBlocks(blockNums)
}

func (s service) GetLatestTxBlock() (*core.TxBlock, error) {
	return s.provider.GetLatestTxBlock()
}

func (s service) GetTransactionsForTxBlock(height uint64) ([][]string, error) {
	txs, err := s.provider.GetTransactionsForTxBlock(fmt.Sprintf("%d", height))
	if err != nil && err.Error() == "-1:TxBlock has no transactions" {
		return [][]string{}, nil
	}

	return txs, err
}

func (s service) GetTxnBodiesForTxBlock(height uint64) ([]core.Transaction, error) {
	txs, err := s.provider.GetTxnBodiesForTxBlock(fmt.Sprintf("%d", height))
	if err != nil && err.Error() == "-1:TxBlock has no transactions" {
		return []core.Transaction{}, nil
	}

	return txs, err
}

func (s service) GetTxnBodiesForTxBlocks(from, count uint64) (txs map[string][]core.Transaction, err error) {
	zap.L().With(zap.Uint64("from", from), zap.Uint64("count", count)).Debug("GetTxnBodiesForTxBlocks")

	blockNums := make([]string, 0)
	for x := from; x < from+count; x++ {
		blockNums = append(blockNums, fmt.Sprintf("%d", x))
	}

	if txs, err = s.provider.GetTxnBodiesForTxBlocks(blockNums); err != nil {
		txs = map[string][]core.Transaction{}
		for _, blockNum := range blockNums {
			blockTxs, err := s.provider.GetTxnBodiesForTxBlock(blockNum)
			if err != nil && err.Error() != "-1:TxBlock has no transactions" {
				return nil, err
			}
			txs[blockNum] = blockTxs
		}
	}

	return
}

func (s service) GetTransaction(hash string) (*core.Transaction, error) {
	return s.provider.GetTransaction(hash)
}

func (s service) GetContractAddressFromTransactionID(txId string) (string, error) {
	return s.provider.GetContractAddressFromTransactionID(txId)
}

func (s service) GetContractAddressFromTransactionIDs(txIds []string) (map[string]string, error) {
	return s.provider.GetContractAddressFromTransactionIDs(txIds)
}

func (s service) GetSmartContractInit(contractAddress string) ([]core.ContractValue, error) {
	return s.provider.GetSmartContractInit(contractAddress)
}

func (s service) GetSmartContractInits(contractAddresses []string) ([][]core.ContractValue, error) {
	return s.provider.GetSmartContractInits(contractAddresses)
}

func (s service) GetSmartContractCode(contractAddress string) (string, error) {
	return s.provider.GetSmartContractCode(contractAddress)
}

func (s service) GetContractState(contractAddress string) (map[string]interface{}, error) {
	resp, err := s.provider.GetSmartContractState(contractAddress)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Result.(map[string]interface{}), err
}

func (s service) GetContractSubState(contractAddress string, params ...interface{}) (string, error) {
	resp, err := s.provider.GetSmartContractSubState(contractAddress, params)
	if err != nil {
		return "", err
	}

	return resp, err
}
