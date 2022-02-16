package zilliqa

import (
	"fmt"
	"go.uber.org/zap"
)

type Service interface {
	GetBlockchainInfo() (*BlockchainInfo, error)
	GetDSBlock(height uint64) (*DSBlock, error)
	GetTxBlock(height uint64) (*TxBlock, error)
	GetTxBlocks(from uint64, count uint) ([]TxBlock, error)
	GetLatestTxBlock() (*TxBlock, error)
	GetTransactionsForTxBlock(height uint64) ([]string, error)
	GetTxnBodiesForTxBlock(height uint64) ([]Transaction, error)
	GetTxnBodiesForTxBlocks(from, count uint64) (map[string][]Transaction, error)
	GetTransaction(hash string) (*Transaction, error)

	GetContractAddressFromTransactionID(txId string) (string, error)
	GetContractAddressFromTransactionIDs(txIds []string) (map[string]string, error)
	GetSmartContractInit(contractAddress string) ([]ContractValue, error)
	GetSmartContractInits(contractAddresses []string) ([][]ContractValue, error)
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

func (s service) GetBlockchainInfo() (*BlockchainInfo, error) {
	return s.provider.GetBlockchainInfo()
}

func (s service) GetDSBlock(height uint64) (*DSBlock, error) {
	return s.provider.GetDsBlock(fmt.Sprintf("%d", height))
}

func (s service) GetTxBlock(height uint64) (*TxBlock, error) {
	return s.provider.GetTxBlock(fmt.Sprintf("%d", height))
}

func (s service) GetTxBlocks(from uint64, count uint) ([]TxBlock, error) {
	blockNums := make([]string, 0)

	for x := from; x < from+uint64(count); x++ {
		blockNums = append(blockNums, fmt.Sprintf("%d", x))
	}

	return s.provider.GetTxBlocks(blockNums)
}

func (s service) GetLatestTxBlock() (*TxBlock, error) {
	return s.provider.GetLatestTxBlock()
}

func (s service) GetTransactionsForTxBlock(blockNum uint64) ([]string, error) {
	addrs := make([]string, 0)

	txBlock, err := s.provider.GetTransactionsForTxBlock(fmt.Sprintf("%d", blockNum))
	if err != nil && err.Error() == "-1:TxBlock has no transactions" {
		zap.L().With(zap.Uint64("height", blockNum)).Warn("TxBlock has no transactions")
		return []string{}, nil
	}

	for _, txs := range txBlock {
		for _, tx := range txs {
			addrs = append(addrs, tx)
		}
	}

	return addrs, err
}

func (s service) GetTxnBodiesForTxBlock(height uint64) ([]Transaction, error) {
	txs, err := s.provider.GetTxnBodiesForTxBlock(fmt.Sprintf("%d", height))
	if err != nil {
		return []Transaction{}, nil
	}

	return txs, err
}

func (s service) GetTxnBodiesForTxBlocks(from, count uint64) (txs map[string][]Transaction, err error) {
	zap.L().With(zap.Uint64("from", from), zap.Uint64("count", count)).Debug("GetTxnBodiesForTxBlocks")

	blockNums := make([]string, 0)
	for x := from; x < from+count; x++ {
		if x == 1664279 {
			// @todo 1664279 returns a  -20:Failed to get Microblock on mainnet. No fucking idea why
			continue
		}
		blockNums = append(blockNums, fmt.Sprintf("%d", x))
	}

	if txs, err = s.provider.GetTxnBodiesForTxBlocks(blockNums); err != nil {
		txs = map[string][]Transaction{}
		for _, blockNum := range blockNums {
			blockTxs, err := s.provider.GetTxnBodiesForTxBlock(blockNum)
			if err != nil {
				return nil, err
			}
			txs[blockNum] = blockTxs
		}
	}

	return txs, nil
}

func (s service) GetTransaction(hash string) (*Transaction, error) {
	return s.provider.GetTransaction(hash)
}

func (s service) GetContractAddressFromTransactionID(txId string) (string, error) {
	return s.provider.GetContractAddressFromTransactionID(txId)
}

func (s service) GetContractAddressFromTransactionIDs(txIds []string) (map[string]string, error) {
	return s.provider.GetContractAddressFromTransactionIDs(txIds)
}

func (s service) GetSmartContractInit(contractAddress string) ([]ContractValue, error) {
	return s.provider.GetSmartContractInit(contractAddress)
}

func (s service) GetSmartContractInits(contractAddresses []string) ([][]ContractValue, error) {
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

	return interface{}(resp.Result).(map[string]interface{}), err
}

func (s service) GetContractSubState(contractAddress string, params ...interface{}) (string, error) {
	resp, err := s.provider.GetSmartContractSubState(contractAddress, params)
	if err != nil {
		return "", err
	}

	return resp, err
}
