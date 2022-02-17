package zilliqa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	jsonrpcVersion = "2.0"
)

// A rpcClient represents a JSON RPC client (over HTTP(s)).
type rpcClient struct {
	url        string
	httpClient *retryablehttp.Client
	timeout    int
	debug      bool
}

// rpcRequest represent a RCP request
type rpcRequest struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      int64       `json:"id"`
	JsonRpc string      `json:"jsonrpc"`
}

type rpcRequests []*rpcRequest

// RPCErrorCode represents an error code to be used as a part of an RPCError
// which is in turn used in a JSON-RPC Response object.
//
// A specific type is used to help ensure the wrong errors aren't used.
type RPCErrorCode int

// RPCError represents an error that is used as a part of a JSON-RPC Response
// object.
type RPCError struct {
	Code    RPCErrorCode `json:"code,omitempty"`
	Message string       `json:"message,omitempty"`
}

// Guarantee RPCError satisfies the builtin error interface.
var _, _ error = RPCError{}, (*RPCError)(nil)

// Error returns a string describing the RPC error.  This satisfies the
// builtin error interface.
func (e RPCError) Error() string {
	return fmt.Sprintf("%d:%s", e.Code, e.Message)
}

type rpcResponse struct {
	Id     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

type rpcResponses []*rpcResponse

func (rResp rpcResponse) ResultAsJson() ([]byte, error) {
	return json.Marshal(rResp.Result)
}

func (rResp rpcResponse) ResultAsString() string {
	return string(rResp.Result)
}

func (rResp rpcResponse) ResultAsFloat64() (float64, error) {
	return json.Number(rResp.Result).Float64()
}

func (rResp rpcResponse) ResultAsInt64() (int64, error) {
	return json.Number(rResp.Result).Int64()
}

func NewClient(url string, timeout int, debug bool) (*rpcClient, error) {
	if len(url) == 0 {
		return nil, errors.New("bad call missing argument host")
	}

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil
	retryClient.RetryMax = 3

	return &rpcClient{
		url,
		retryClient,
		timeout,
		debug,
	}, nil
}

func NewRequest(method string, params ...interface{}) *rpcRequest{
	return &rpcRequest{method, params, time.Now().UnixNano(), jsonrpcVersion}
}

// doTimeoutRequest process a HTTP request with timeout
func (c *rpcClient) doTimeoutRequest(timer *time.Timer, req *retryablehttp.Request) (*http.Response, error) {
	type result struct {
		resp *http.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := c.httpClient.Do(req)
		done <- result{resp, err}
	}()
	// Wait for the read or the timeout
	select {
	case r := <-done:
		return r.resp, r.err
	case <-timer.C:
		return nil, errors.New("timeout reading data from server")
	}
}

// call prepare & exec the request
func (c *rpcClient) call(method string, params interface{}) (rr *rpcResponse, err error) {
	rpcR := rpcRequest{method, params, time.Now().UnixNano(), jsonrpcVersion}
	payloadBuffer := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(payloadBuffer)
	err = jsonEncoder.Encode(rpcR)
	if err != nil {
		return
	}

	zap.L().With(zap.String("request", rpcR.Method), zap.String("params", fmt.Sprintf("%v", params))).Debug("Zilliqa: RPC Request")
	if c.debug {
		zap.L().With(zap.String("request", payloadBuffer.String())).Debug("Zilliqa: RPC Request")
	}

	req, err := retryablehttp.NewRequest("POST", c.url, payloadBuffer)
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Accept", "application/json")

	resp, err := c.doTimeoutRequest(time.NewTimer(time.Duration(c.timeout)*time.Second), req)
	if err != nil {
		zap.L().With(zap.Error(err)).Warn("Zilliqa: RPC Failure")
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if c.debug {
		zap.L().With(zap.String("response", string(data))).Debug("Zilliqa: RPC Response")
	}

	err = json.Unmarshal(data, &rr)
	return
}


func (c *rpcClient) callBatch(requests rpcRequests) (rr rpcResponses, err error) {
	if len(requests) == 0 {
		return nil, errors.New("empty request list")
	}

	for i, req := range requests {
		req.Id = int64(i)
		req.JsonRpc = jsonrpcVersion
	}
	payloadBuffer := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(payloadBuffer)
	err = jsonEncoder.Encode(requests)
	if err != nil {
		return
	}

	zap.L().With(zap.String("request", requests[0].Method), zap.Int("count", len(requests))).Debug("Zilliqa: RPC Batch Request")
	if c.debug {
		zap.L().With(zap.String("request", payloadBuffer.String())).Debug("Zilliqa: RPC Request")
	}
	req, err := retryablehttp.NewRequest("POST", c.url, payloadBuffer)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Accept", "application/json")

	resp, err := c.doTimeoutRequest(time.NewTimer(time.Duration(c.timeout)*time.Second), req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if c.debug {
		zap.L().With(zap.String("response", string(data))).Debug("Zilliqa: RPC Response")
	}

	err = json.Unmarshal(data, &rr)
	return
}