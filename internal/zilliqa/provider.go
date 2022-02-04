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
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

type Provider struct {
	host      string
	rpcClient *rpcClient
}

func NewProvider(rpcClient *rpcClient) *Provider {
	return &Provider{rpcClient: rpcClient}
}

func (p *Provider) GetNetworkId() (string, error) {
	response, err := p.call("GetNetworkId")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetBlockchainInfo() (*BlockchainInfo, error) {
	response, err := p.call("GetBlockchainInfo")
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var blockchainInfo BlockchainInfo
	if err := json.Unmarshal(jsonString, &blockchainInfo); err != nil {
		return nil, err
	}

	return &blockchainInfo, nil
}

func (p *Provider) GetShardingStructure() (*ShardingStructure, error) {
	response, err := p.call("GetShardingStructure")
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var shardingStructure ShardingStructure
	if err := json.Unmarshal(jsonString, &shardingStructure); err != nil{
		return nil, err
	}

	return &shardingStructure, nil

}

func (p *Provider) GetDsBlock(blockNum string) (*DSBlock, error) {
	result, err := p.call("GetDsBlock", blockNum)
	if err != nil {
		return nil, err
	}

	jsonString, err := result.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var dsBlock DSBlock
	if err := json.Unmarshal(jsonString, &dsBlock); err != nil {
		return nil, err
	}

	return &dsBlock, nil
}

func (p *Provider) GetLatestDsBlock() (*DSBlock, error) {
	result, err := p.call("GetLatestDsBlock")
	if err != nil {
		return nil, err
	}

	jsonString, err := result.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var dsBlock DSBlock
	if err := json.Unmarshal(jsonString, &dsBlock); err != nil {
		return nil, err
	}

	return &dsBlock, nil
}

func (p *Provider) GetNumDSBlocks() (string, error) {
	response, err := p.call("GetNumDSBlocks")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetDSBlockRate() (float64, error) {
	result, err := p.call("GetDSBlockRate")
	if err != nil {
		return 0, err
	}

	return json.Number(result.Result).Float64()
}

func (p *Provider) DSBlockListing(dsBlockListing int) (*BlockList, error) {
	response, err := p.call("DSBlockListing", dsBlockListing)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var list BlockList
	if err := json.Unmarshal(jsonString, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

func (p *Provider) GetTxBlock(blockNum string) (*TxBlock, error) {
	response, err := p.call("GetTxBlock", blockNum)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var txBlock TxBlock
	if err := json.Unmarshal(jsonString, &txBlock); err != nil {
		return nil, err
	}

	return &txBlock, nil
}

func (p *Provider) GetTxBlocks(blockNums []string) ([]TxBlock, error) {
	var requests rpcRequests
	for _, blockNum := range blockNums {
		r := NewRequest("GetTxBlock", blockNum)
		requests = append(requests, r)
	}

	responses, err := p.callBatch(requests)
	if err != nil {
		return nil, err
	}

	var txBlocks []TxBlock

	for _, response := range responses {
		jsonString, err := response.ResultAsJson()
		if err != nil {
			return nil, err
		}

		var txBlock TxBlock
		if err := json.Unmarshal(jsonString, &txBlock); err != nil {
			return nil, err
		}

		txBlocks = append(txBlocks, txBlock)
	}

	return txBlocks, nil
}

func (p *Provider) GetLatestTxBlock() (*TxBlock, error) {
	response, err := p.call("GetLatestTxBlock")
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var txBlock TxBlock
	if err := json.Unmarshal(jsonString, &txBlock); err != nil {
		return nil, err
	}

	return &txBlock, nil
}

func (p *Provider) GetNumTxBlocks() (string, error) {
	response, err := p.call("GetNumTxBlocks")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetTxBlockRate() (float64, error) {
	response, err := p.call("GetTxBlockRate")
	if err != nil {
		return 0, err
	}

	return response.ResultAsFloat64()
}

func (p *Provider) TxBlockListing(page int) (*BlockList, error) {
	response, err := p.call("TxBlockListing", page)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var list BlockList
	if err := json.Unmarshal(jsonString, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

func (p *Provider) GetNumTransactions() (string, error) {
	response, err := p.call("GetNumTransactions")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetTransactionRate() (float64, error) {
	response, err := p.call("GetTransactionRate")
	if err != nil {
		return 0, err
	}

	return response.ResultAsFloat64()
}

func (p *Provider) GetCurrentMiniEpoch() (string, error) {
	response, err := p.call("GetCurrentMiniEpoch")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetCurrentDSEpoch() (string, error) {
	response, err := p.call("GetCurrentDSEpoch")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetPrevDifficulty() (int64, error) {
	response, err := p.call("GetPrevDifficulty")
	if err != nil {
		return 0, err
	}

	return response.ResultAsInt64()
}

func (p *Provider) GetPrevDSDifficulty() (int64, error) {
	response, err := p.call("GetPrevDSDifficulty")
	if err != nil {
		return 0, err
	}

	return response.ResultAsInt64()
}

func (p *Provider) GetTotalCoinSupply() (string, error) {
	response, err := p.call("GetTotalCoinSupply")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetMinerInfo(dsNumber string) (*MinerInfo, error) {
	response, err := p.call("GetMinerInfo", dsNumber)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var minerInfo MinerInfo
	if err := json.Unmarshal(jsonString, &minerInfo); err != nil {
		return nil, err
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
func (p *Provider) GetPendingTxn(tx string) (*rpcResponse, error) {
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
func (p *Provider) GetPendingTxns() (*rpcResponse, error) {
	return p.call("GetPendingTxns")
}

func (p *Provider) GetTransaction(txId string) (*Transaction, error) {
	response, err := p.call("GetTransaction", txId)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var transaction Transaction
	if err := json.Unmarshal(jsonString, &transaction); err != nil {
		return nil, err
	}

	return &transaction, nil
}

func (p *Provider) GetTransactionBatch(txIds []string) ([]*Transaction, error) {
	var requests rpcRequests
	for _, hash := range txIds {
		r := NewRequest("GetTransaction", []string{hash})
		requests = append(requests, r)
	}

	responses, err := p.rpcClient.callBatch(requests)
	if err != nil {
		return nil, err
	}

	var transactions []*Transaction

	for _, response := range responses {
		jsonString, err := response.ResultAsJson()
		if err != nil {
			return nil, err
		}

		var transaction Transaction
		if err := json.Unmarshal(jsonString, &transaction); err != nil {
			return nil, err
		}

		transactions = append(transactions, &transaction)
	}

	return transactions, nil

}

func (p *Provider) GetRecentTransactions() (*Transactions, error) {
	response, err := p.call("GetRecentTransactions")
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var transactions Transactions
	if err := json.Unmarshal(jsonString, &transactions); err != nil {
		return nil, err
	}

	return &transactions, nil
}

func (p *Provider) GetTransactionsForTxBlock(blockNum string) ([][]string, error) {
	response, err := p.call("GetTransactionsForTxBlock", blockNum)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var transactions [][]string
	if err := json.Unmarshal(jsonString, &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (p *Provider) GetTxnBodiesForTxBlock(blockNum string) ([]Transaction, error) {
	response, err := p.call("GetTxnBodiesForTxBlock", blockNum)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var transactions []Transaction
	if err := json.Unmarshal(jsonString, &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (p *Provider) GetTxnBodiesForTxBlocks(blockNums []string) (map[string][]Transaction, error) {
	var requests rpcRequests
	for _, blockNum := range blockNums {
		r := NewRequest("GetTxnBodiesForTxBlock", blockNum)
		requests = append(requests, r)
	}

	responses, err := p.callBatch(requests)
	if err != nil {
		return nil, err
	}

	transactions := map[string][]Transaction{}

	for idx, response := range responses {
		if response.Error != nil {
			if response.Error.Message == "TxBlock has no transactions" {
				continue
			}
			if response.Error.Message == "Txn Hash not Present" {
				continue
			}
			return nil, response.Error
		}

		var txs []Transaction
		jsonString, err := response.ResultAsJson()
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(jsonString, &txs); err != nil {
			return nil, err
		}

		transactions[blockNums[idx]] = txs
	}

	return transactions, nil
}

func (p *Provider) GetNumTxnsTxEpoch() (string, error) {
	response, err := p.call("GetNumTxnsTxEpoch")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetNumTxnsDSEpoch() (string, error) {
	response, err := p.call("GetNumTxnsDSEpoch")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetMinimumGasPrice() (string, error) {
	response, err := p.call("GetMinimumGasPrice")
	if err != nil {
		return "", err
	}

	return response.ResultAsString(), nil
}

func (p *Provider) GetSmartContractCode(contractAddr string) (string, error) {
	result, err := p.call("GetSmartContractCode", contractAddr)
	if err != nil {
		return "", err
	}

	if resultMap, ok := interface{}(result.Result).(map[string]interface{}); ok {
		if code, ok := resultMap["code"]; ok {
			return code.(string), nil
		}
	}

	return "", errors.New("failed to get code for contract")
}

func (p *Provider) GetSmartContractInit(contractAddr string) ([]ContractValue, error) {
	response, err := p.call("GetSmartContractInit", contractAddr)
	if err != nil {
		return nil, err
	}

	jsonString, err := response.ResultAsJson()
	if err != nil {
		return nil, err
	}

	var init []ContractValue
	if err := json.Unmarshal(jsonString, &init); err != nil {
		return nil, err
	}

	return init, nil
}

func (p *Provider) GetSmartContractInits(contractAddresses []string) ([][]ContractValue, error) {
	var requests rpcRequests
	for _, contractAddress := range contractAddresses {
		r := NewRequest("GetSmartContractInit", contractAddress)
		requests = append(requests, r)
	}

	responses, err := p.rpcClient.callBatch(requests)
	if err != nil {
		return nil, err
	}

	contractValues := make([][]ContractValue, 0)

	for _, response := range responses {
		var contractValue []ContractValue
		jsonString, _ := response.ResultAsJson()
		_ = json.Unmarshal(jsonString, &contractValue)

		contractValues = append(contractValues, contractValue)
	}

	return contractValues, nil
}

func (p *Provider) GetSmartContractState(contractAddr string) (*rpcResponse, error) {
	return p.call("GetSmartContractState", contractAddr)
}

func (p *Provider) GetSmartContractSubState(contractAddr string, params ...interface{}) (string, error) {
	//we should hack here for now
	type req struct {
		Id      string      `json:"id"`
		Jsonrpc string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}

	ps := []interface{}{
		contractAddr,
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
	request, err := http.NewRequest("POST", p.rpcClient.url, reader)
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
func (p *Provider) GetSmartContracts(address string) (*rpcResponse, error) {
	return p.call("GetSmartContracts", address)
}

// Returns a smart contract address of 20 bytes. This is represented as a String.
// NOTE: This only works for contract deployment transactions.
func (p *Provider) GetContractAddressFromTransactionID(txId string) (string, error) {
	zap.S().Infof("GetContractAddressFromTransactionID(%s)", txId)
	result, err := p.call("GetContractAddressFromTransactionID", txId)
	if err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", result.Error
	}

	return string(result.Result), nil
}

// Returns a smart contract address of 20 bytes. This is represented as a String.
// NOTE: This only works for contract deployment transactions.
func (p *Provider) GetContractAddressFromTransactionIDs(transactionIds []string) (map[string]string, error) {
	if len(transactionIds) == 0 {
		return map[string]string{}, nil
	}

	var requests rpcRequests
	for _, transactionId := range transactionIds {
		r := NewRequest("GetContractAddressFromTransactionID", transactionId)
		requests = append(requests, r)
	}

	results, err := p.callBatch(requests)
	if err != nil {
		return nil, err
	}

	contractAddresses := map[string]string{}
	for idx, result := range results {
		if result.Error == nil {
			contractAddresses[transactionIds[idx]] = string(result.Result)
		} else {
			contractAddresses[transactionIds[idx]] = ""
		}
	}

	return contractAddresses, nil
}

// Returns the current balance of an account, measured in the smallest accounting unit Qa (or 10^-12 Zil).
// This is represented as a String
// Returns the current nonce of an account. This is represented as an Number.
func (p *Provider) GetBalance(address string) (*BalanceAndNonce, error) {
	result, err := p.call("GetBalance", address)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	balanceAndNonce := BalanceAndNonce{
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

func (p *Provider) call(method string, params ...interface{}) (*rpcResponse, error) {
	response, err := p.rpcClient.call(method, params)

	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.New("rpc response is nil, please check your network status")
	}

	if response.Error != nil {
		return nil, response.Error
	}

	return response, nil
}

func (p *Provider) callBatch(requests rpcRequests) (rpcResponses, error) {
	responses, err := p.rpcClient.callBatch(requests)

	if err != nil {
		return nil, err
	}

	if responses == nil {
		return nil, errors.New("rpc response is nil, please check your network status")
	}

	return responses, nil
}
