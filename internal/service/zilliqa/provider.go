/*
 * Copyright (C) 2019 Zilliqa
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
package zilliqa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Zilliqa/gozilliqa-sdk/core"
	"github.com/Zilliqa/gozilliqa-sdk/provider"
	"github.com/ybbus/jsonrpc"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
)

type Provider struct {
	host      string
	rpcClient jsonrpc.RPCClient
}

func NewProvider(host string) *Provider {
	rpcClient := jsonrpc.NewClient(host)
	return &Provider{host: host, rpcClient: rpcClient}
}

// Returns the CHAIN_ID of the specified network. This is represented as a String.
func (p *Provider) GetNetworkId() (string, error) {
	result, err := p.call("GetNetworkId")
	if err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", result.Error
	}
	return result.Result.(string), nil
}

// Returns the current network statistics for the specified network.
func (p *Provider) GetBlockchainInfo() (*core.BlockchainInfo, error) {
	result, err := p.call("GetBlockchainInfo")
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	var blockchainInfo core.BlockchainInfo
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &blockchainInfo)
	if err3 != nil {
		return nil, err3
	}

	return &blockchainInfo, nil

}

func (p *Provider) GetShardingStructure() (*core.ShardingStructure, error) {
	result, err := p.call("GetShardingStructure")
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var shardingStructure core.ShardingStructure
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &shardingStructure)
	if err3 != nil {
		return nil, err3
	}

	return &shardingStructure, nil

}

// Returns the details of a specified Directory Service block.
func (p *Provider) GetDsBlock(block_number string) (*core.DSBlock, error) {
	result, err := p.call("GetDsBlock", block_number)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var dsBlock core.DSBlock

	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &dsBlock)
	if err3 != nil {
		return nil, err3
	}

	return &dsBlock, nil
}

// Returns the details of the most recent Directory Service block.
func (p *Provider) GetLatestDsBlock() (*core.DSBlock, error) {
	result, err := p.call("GetLatestDsBlock")
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var dsBlock core.DSBlock

	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &dsBlock)
	if err3 != nil {
		return nil, err3
	}

	return &dsBlock, nil
}

// Returns the current number of validated Directory Service blocks in the network.
// This is represented as a String.
func (p *Provider) GetNumDSBlocks() (string, error) {
	result, err := p.call("GetNumDSBlocks")
	if err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", result.Error
	}
	return result.Result.(string), nil
}

// Returns the current Directory Service blockrate per second.
func (p *Provider) GetDSBlockRate() (float64, error) {
	result, err := p.call("GetDSBlockRate")
	if err != nil {
		return 0, err
	}

	if result.Error != nil {
		return 0, result.Error
	}

	rate, err2 := result.Result.(json.Number).Float64()
	if err2 != nil {
		return 0, err2
	}

	return rate, nil
}

// Returns a paginated list of up to 10 Directory Service (DS) blocks and their block hashes for a specified page.
// The maxPages variable that specifies the maximum number of pages available is also returned.
func (p *Provider) DSBlockListing(ds_block_listing int) (*core.BlockList, error) {
	result, err := p.call("DSBlockListing", ds_block_listing)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var list core.BlockList
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &list)
	if err3 != nil {
		return nil, err3
	}

	return &list, nil
}

// Returns the details of a specified Transaction block.
func (p *Provider) GetTxBlock(tx_block string) (*core.TxBlock, error) {
	result, err := p.call("GetTxBlock", tx_block)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var txBlock core.TxBlock

	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &txBlock)
	if err3 != nil {
		return nil, err3
	}

	return &txBlock, nil
}

// Returns the details of a specified Transaction block.
func (p *Provider) GetTxBlocks(blockNums []string) ([]core.TxBlock, error) {
	var requests jsonrpc.RPCRequests
	for _, blockNum := range blockNums {
		r := jsonrpc.NewRequest("GetTxBlock", blockNum)
		requests = append(requests, r)
	}

	zap.L().With(
		zap.String("from", blockNums[0]),
		zap.String("to", blockNums[len(blockNums)-1]),
	).Info("GetTxBlock")

	results, err := p.callBatch(requests)
	if err != nil {
		return nil, err
	}

	var txBlocks []core.TxBlock

	for _, result := range results {
		var txBlock core.TxBlock
		jsonString, err2 := json.Marshal(result.Result)
		if err2 != nil {
			return txBlocks, err2
		}
		err3 := json.Unmarshal(jsonString, &txBlock)
		if err3 != nil {
			return txBlocks, err3
		}

		txBlocks = append(txBlocks, txBlock)
	}

	return txBlocks, nil
}

// Returns the details of the most recent Transaction block.
func (p *Provider) GetLatestTxBlock() (*core.TxBlock, error) {
	result, err := p.call("GetLatestTxBlock")
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var txBlock core.TxBlock

	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &txBlock)
	if err3 != nil {
		return nil, err3
	}

	return &txBlock, nil
}

// Returns the current number of Transaction blocks in the network.
// This is represented as a String.
func (p *Provider) GetNumTxBlocks() (string, error) {
	result, err := p.call("GetNumTxBlocks")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the current Transaction blockrate per second for the network.
func (p *Provider) GetTxBlockRate() (float64, error) {
	result, err := p.call("GetTxBlockRate")
	if err != nil {
		return 0, err
	}

	if result.Error != nil {
		return 0, result.Error
	}

	rate, err2 := result.Result.(json.Number).Float64()
	if err2 != nil {
		return 0, err2
	}

	return rate, nil
}

// Returns a paginated list of up to 10 Transaction blocks and their block hashes for a specified page.
// The maxPages variable that specifies the maximum number of pages available is also returned.
func (p *Provider) TxBlockListing(page int) (*core.BlockList, error) {
	result, err := p.call("TxBlockListing", page)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var list core.BlockList
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &list)
	if err3 != nil {
		return nil, err3
	}

	return &list, nil
}

// Returns the current number of validated Transactions in the network.
// This is represented as a String.
func (p *Provider) GetNumTransactions() (string, error) {
	result, err := p.call("GetNumTransactions")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the current Transaction rate per second (TPS) of the network.
// This is represented as an Number.
func (p *Provider) GetTransactionRate() (float64, error) {
	result, err := p.call("GetTransactionRate")

	if err != nil {
		return 0, err
	}

	if result.Error != nil {
		return 0, result.Error
	}

	rate, err2 := result.Result.(json.Number).Float64()
	if err2 != nil {
		return 0, err2
	}

	return rate, nil
}

// Returns the current TX block number of the network.
// This is represented as a String.
func (p *Provider) GetCurrentMiniEpoch() (string, error) {
	result, err := p.call("GetCurrentMiniEpoch")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the current number of DS blocks in the network.
// This is represented as a String.
func (p *Provider) GetCurrentDSEpoch() (string, error) {
	result, err := p.call("GetCurrentDSEpoch")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the minimum shard difficulty of the previous block.
// This is represented as an Number.
func (p *Provider) GetPrevDifficulty() (int64, error) {
	result, err := p.call("GetPrevDifficulty")
	if err != nil {
		return 0, err
	}

	if result.Error != nil {
		return 0, result.Error
	}

	difficulty, err2 := result.Result.(json.Number).Int64()
	if err2 != nil {
		return 0, err2
	}

	return difficulty, nil
}

// Returns the minimum DS difficulty of the previous block.
// This is represented as an Number.
func (p *Provider) GetPrevDSDifficulty() (int64, error) {
	result, err := p.call("GetPrevDSDifficulty")
	if err != nil {
		return 0, err
	}

	if result.Error != nil {
		return 0, result.Error
	}

	difficulty, err2 := result.Result.(json.Number).Int64()
	if err2 != nil {
		return 0, err2
	}

	return difficulty, nil
}

// Returns the total supply (ZIL) of coins in the network. This is represented as a String.
func (p *Provider) GetTotalCoinSupply() (string, error) {
	result, err := p.call("GetTotalCoinSupply")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the mining nodes (i.e., the members of the DS committee and shards) at the specified DS block.
// Notes: 1. Nodes owned by Zilliqa Research are omitted. 2. dscommittee has no size field since the DS committee size
// is fixed for a given chain. 3. For the Zilliqa Mainnet, this API is only available from DS block 5500 onwards.
func (p *Provider) GetMinerInfo(dsNumber string) (*core.MinerInfo, error) {
	result, err := p.call("GetMinerInfo", dsNumber)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var minerInfo core.MinerInfo
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &minerInfo)
	if err3 != nil {
		return nil, err3
	}

	return &minerInfo, nil
}

// Returns the pending status of a specified Transaction. Possible results are:
//
//  confirmed	code	info
//  false	0	Txn not pending
//  false	1	Nonce too high
//  false	2	Could not fit in as microblock gas limit reached
//  false	3	Transaction valid but consensus not reached
func (p *Provider) GetPendingTxn(tx string) (*jsonrpc.RPCResponse, error) {
	return p.call("GetPendingTxn", tx)
}

// Returns the pending status of all unvalidated Transactions.
//
//  For each entry, the possible results are:
//
//  confirmed	code	info
//  false	0	Txn not pending
//  false	1	Nonce too high
//  false	2	Could not fit in as microblock gas limit reached
//  false	3	Transaction valid but consensus not reached
func (p *Provider) GetPendingTxns() (*jsonrpc.RPCResponse, error) {
	return p.call("GetPendingTxns")
}

// Create a new Transaction object and send it to the network to be process.
func (p *Provider) CreateTransaction(payload provider.TransactionPayload) (*jsonrpc.RPCResponse, error) {
	return p.call("CreateTransaction", &payload)
}

func (p *Provider) CreateTransactionBatch(payloads [][]provider.TransactionPayload) (jsonrpc.RPCResponses, error) {
	var requests jsonrpc.RPCRequests
	for _, payload := range payloads {
		r := jsonrpc.NewRequest("CreateTransaction", payload)
		requests = append(requests, r)
	}
	return p.rpcClient.CallBatch(requests)
}

func (p *Provider) CreateTransactionRaw(payload []byte) (*jsonrpc.RPCResponse, error) {
	var pl provider.TransactionPayload
	err := json.Unmarshal(payload, &pl)
	if err != nil {
		panic(err.Error())
	}
	return p.call("CreateTransaction", &pl)
}

// Returns the details of a specified Transaction.
// Note: If the transaction had an data field or code field, it will be displayed
func (p *Provider) GetTransaction(transaction_hash string) (*core.Transaction, error) {
	result, err := p.call("GetTransaction", transaction_hash)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var transaction core.Transaction
	jsonString, err2 := json.Marshal(result.Result)
	log.Println(string(jsonString))
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &transaction)
	if err3 != nil {
		return nil, err3
	}

	return &transaction, nil
}

func (p *Provider) GetTransactionBatch(transactionHashes []string) ([]*core.Transaction, error) {
	var requests jsonrpc.RPCRequests
	for _, hash := range transactionHashes {
		r := jsonrpc.NewRequest("GetTransaction", []string{hash})
		requests = append(requests, r)
	}

	results, err := p.rpcClient.CallBatch(requests)
	if err != nil {
		return nil, err
	}

	var transactions []*core.Transaction

	for _, result := range results {
		var transaction core.Transaction
		jsonString, err2 := json.Marshal(result.Result)
		if err2 != nil {
			return transactions, err2
		}
		err3 := json.Unmarshal(jsonString, &transaction)
		if err3 != nil {
			return transactions, err3
		}

		transactions = append(transactions, &transaction)
	}

	return transactions, nil

}

// Returns the most recent 100 transactions that are validated by the Zilliqa network.
func (p *Provider) GetRecentTransactions() (*core.Transactions, error) {
	result, err := p.call("GetRecentTransactions")
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var transactions core.Transactions
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &transactions)
	if err3 != nil {
		return nil, err3
	}

	return &transactions, nil
}

// Returns the validated transactions included within a specfied final transaction block as an array of length i,
// where i is the number of shards plus the DS committee. The transactions are grouped based on the group that processed
// the transaction. The first element of the array refers to the first shard. The last element of the array at index, i,
// refers to the transactions processed by the DS Committee.
func (p *Provider) GetTransactionsForTxBlock(tx_block_number string) ([][]string, error) {
	result, err := p.call("GetTransactionsForTxBlock", tx_block_number)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var transactions [][]string
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &transactions)
	if err3 != nil {
		return nil, err3
	}

	return transactions, nil
}

func (p *Provider) GetTxnBodiesForTxBlock(tx_block_number string) ([]core.Transaction, error) {
	result, err := p.call("GetTxnBodiesForTxBlock", tx_block_number)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var transactions []core.Transaction
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &transactions)
	if err3 != nil {
		return nil, err3
	}

	return transactions, nil
}

func (p *Provider) GetTxnBodiesForTxBlocks(blockNums []string) (map[string][]core.Transaction, error) {
	var requests jsonrpc.RPCRequests
	for _, blockNum := range blockNums {
		r := jsonrpc.NewRequest("GetTxnBodiesForTxBlock", blockNum)
		requests = append(requests, r)
	}

	zap.L().With(
		zap.String("from", blockNums[0]),
		zap.String("to", blockNums[len(blockNums)-1]),
	).Debug("GetTxnBodiesForTxBlocks")

	results, err := p.callBatch(requests)
	if err != nil {
		return nil, err
	}

	transactions := map[string][]core.Transaction{}

	for idx, result := range results {
		if result.Error != nil {
			zap.L().With(
				zap.String("data", fmt.Sprintf("%v", result.Error.Data)),
				zap.Int("code", result.Error.Code),
				zap.String("height", blockNums[idx]),
			).Debug(result.Error.Message)

			if result.Error.Message == "TxBlock has no transactions" {
				continue
			}
			if result.Error.Message == "Txn Hash not Present" {
				continue
			}
			return nil, result.Error
		}

		var txs []core.Transaction
		jsonString, err2 := json.Marshal(result.Result)
		if err2 != nil {
			return nil, err2
		}

		err3 := json.Unmarshal(jsonString, &txs)
		if err3 != nil {
			return nil, err3
		}

		transactions[blockNums[idx]] = txs
	}

	return transactions, nil
}

// Returns the number of validated transactions included in this Transaction epoch.
// This is represented as String.
func (p *Provider) GetNumTxnsTxEpoch() (string, error) {
	result, err := p.call("GetNumTxnsTxEpoch")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the number of validated transactions included in this DS epoch.
// This is represented as String.
func (p *Provider) GetNumTxnsDSEpoch() (string, error) {
	result, err := p.call("GetNumTxnsDSEpoch")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns the minimum gas price for this DS epoch, measured in the smallest price unit Qa (or 10^-12 Zil) in Zilliqa.
// This is represented as a String.
func (p *Provider) GetMinimumGasPrice() (string, error) {
	result, err := p.call("GetMinimumGasPrice")
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil

}

// Returns the Scilla code associated with a smart contract address.
// This is represented as a String.
func (p *Provider) GetSmartContractCode(contract_address string) (string, error) {
	result, err := p.call("GetSmartContractCode", contract_address)
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	if resultMap, ok := result.Result.(map[string]interface{}); ok {
		if code, ok := resultMap["code"]; ok {
			return code.(string), nil
		}
	}

	return "", errors.New("failed to get code for contract")
}

// Returns the initialization (immutable) parameters of a given smart contract, represented in a JSON format.
func (p *Provider) GetSmartContractInit(contract_address string) ([]core.ContractValue, error) {
	result, err := p.call("GetSmartContractInit", contract_address)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	var init []core.ContractValue
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &init)
	if err3 != nil {
		return nil, err3
	}

	return init, nil
}

// Returns the initialization (immutable) parameters of a given smart contract, represented in a JSON format.
func (p *Provider) GetSmartContractInits(contractAddresses []string) ([][]core.ContractValue, error) {
	var requests jsonrpc.RPCRequests
	for _, contractAddress := range contractAddresses {
		r := jsonrpc.NewRequest("GetSmartContractInit", contractAddress)
		requests = append(requests, r)
	}

	results, err := p.rpcClient.CallBatch(requests)
	if err != nil {
		return nil, err
	}

	contractValues := make([][]core.ContractValue, 0)

	for _, result := range results {
		var contractValue []core.ContractValue
		jsonString, _ := json.Marshal(result.Result)
		_ = json.Unmarshal(jsonString, &contractValue)

		contractValues = append(contractValues, contractValue)
	}

	return contractValues, nil
}

// Returns the state (mutable) variables of a smart contract address, represented in a JSON format.
func (p *Provider) GetSmartContractState(contract_address string) (*jsonrpc.RPCResponse, error) {
	return p.call("GetSmartContractState", contract_address)
}

// Returns the state (or a part specified) of a smart contract address, represented in a JSON format.
func (p *Provider) GetSmartContractSubState(contractAddress string, params ...interface{}) (string, error) {
	//we should hack here for now
	type req struct {
		Id      string      `json:"id"`
		Jsonrpc string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}

	ps := []interface{}{
		contractAddress,
	}

	for _, v := range params {
		ps = append(ps, v)
	}

	r := &req{
		Id:      "1",
		Jsonrpc: "2.0",
		Method:  "GetSmartContractSubState",
		Params:  ps,
	}

	b, _ := json.Marshal(r)
	reader := bytes.NewReader(b)
	request, err := http.NewRequest("POST", p.host, reader)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(result), nil

}

// Returns the list of smart contract addresses created by an User's account and the contracts' latest states.
func (p *Provider) GetSmartContracts(user_address string) (*jsonrpc.RPCResponse, error) {
	return p.call("GetSmartContracts", user_address)
}

// Returns a smart contract address of 20 bytes. This is represented as a String.
// NOTE: This only works for contract deployment transactions.
func (p *Provider) GetContractAddressFromTransactionID(transaction_id string) (string, error) {
	zap.S().Infof("GetContractAddressFromTransactionID(%s)", transaction_id)
	result, err := p.call("GetContractAddressFromTransactionID", transaction_id)
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return result.Result.(string), nil
}

// Returns a smart contract address of 20 bytes. This is represented as a String.
// NOTE: This only works for contract deployment transactions.
func (p *Provider) GetContractAddressFromTransactionIDs(transactionIds []string) (map[string]string, error) {
	if len(transactionIds) == 0 {
		return map[string]string{}, nil
	}

	var requests jsonrpc.RPCRequests
	for _, transactionId := range transactionIds {
		r := jsonrpc.NewRequest("GetContractAddressFromTransactionID", transactionId)
		requests = append(requests, r)
	}

	results, err := p.callBatch(requests)
	if err != nil {
		return nil, err
	}

	contractAddresses := map[string]string{}
	for idx, result := range results {
		if result.Error == nil {
			contractAddresses[transactionIds[idx]] = result.Result.(string)
		} else {
			contractAddresses[transactionIds[idx]] = ""
		}
	}

	return contractAddresses, nil
}

// Returns the current balance of an account, measured in the smallest accounting unit Qa (or 10^-12 Zil).
// This is represented as a String
// Returns the current nonce of an account. This is represented as an Number.
func (p *Provider) GetBalance(user_address string) (*core.BalanceAndNonce, error) {
	result, err := p.call("GetBalance", user_address)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	balanceAndNonce := core.BalanceAndNonce{
		Balance: "0",
		Nonce:   0,
	}
	jsonString, err2 := json.Marshal(result.Result)
	if err2 != nil {
		return nil, err2
	}

	err3 := json.Unmarshal(jsonString, &balanceAndNonce)
	if err3 != nil {
		return nil, err3
	}

	return &balanceAndNonce, nil
}

func (p *Provider) call(method_name string, params ...interface{}) (*jsonrpc.RPCResponse, error) {
	response, err := p.rpcClient.Call(method_name, params)

	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.New("rpc response is nil, please check your network status")
	}

	return response, nil
}

func (p *Provider) callBatch(requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	responses, err := p.rpcClient.CallBatch(requests)

	if err != nil {
		return nil, err
	}

	if responses == nil {
		return nil, errors.New("rpc response is nil, please check your network status")
	}

	return responses, nil
}
