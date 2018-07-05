// Copyright 2018 karma.run AG. All rights reserved.
package main // import "github.com/karmarun/karma.link/link"

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/karmarun/karma.link/abi"
	"github.com/karmarun/karma.link/ast"
	"github.com/karmarun/karma.link/ast/extract"
	"github.com/karmarun/karma.link/auth"
	"github.com/karmarun/karma.link/auth/fs"
	"github.com/karmarun/karma.link/config"
	"github.com/karmarun/karma.link/types"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultGasLimit = 90000

var signer = ethtypes.HomesteadSigner{ethtypes.FrontierSigner{}}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzip *gzip.Writer
}

func (w gzipResponseWriter) Write(bs []byte) (int, error) {
	return w.gzip.Write(bs)
}

var (
	EthClient *ethrpc.Client
)

func main() {

	flag.Parse()

	if config.CombinedJSONPath == "" {
		log.Fatalln("Please specify --combined-json flag. See --help.")
	}

	if config.GethRPCURL == "" {
		log.Fatalln("Please specify --geth-rpc flag. See --help.")
	}

	if config.FSAuthDirectory != "" {
		auth.RegisterAuthenticator(`fs`, fs.Folder(config.FSAuthDirectory))
	}

	{
		c, e := ethrpc.Dial(config.GethRPCURL)
		if e != nil {
			log.Fatalln(e)
		}
		defer c.Close()
		EthClient = c
	}

	file, e := os.Open(config.CombinedJSONPath)
	if e != nil {
		log.Fatalln(e)
	}
	defer file.Close()
	bs, e := ioutil.ReadAll(file)
	if e != nil {
		log.Fatalln(e)
	}
	combined := ast.Combined{}
	if e := json.Unmarshal(bs, &combined); e != nil {
		log.Fatalln(e)
	}
	project := extract.Project(combined)

	rpcServer := rpc.NewServer()

	if e := rpcServer.RegisterName("v1", RpcHandler{project}); e != nil {
		log.Fatalln(e)
	}

	httpServer := http.Server{
		Addr: config.HttpBind,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, rq *http.Request) {
			if strings.Contains(rq.Header.Get(http.CanonicalHeaderKey(`accept-encoding`)), `gzip`) {
				gz, _ := gzip.NewWriterLevel(rw, gzip.BestSpeed)
				rw = gzipResponseWriter{rw, gz}
				rw.Header().Set(http.CanonicalHeaderKey(`content-encoding`), `gzip`)
				defer gz.Close()
			}
			rw.Header().Set(http.CanonicalHeaderKey(`content-type`), `application/json; charset=UTF-8`)
			rpcServer.ServeCodec(
				jsonrpc.NewServerCodec(struct {
					io.Writer
					io.ReadCloser
				}{
					rw,
					rq.Body,
				}),
			)
		}),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second * 2,
		WriteTimeout:      time.Second * 3,
		IdleTimeout:       time.Second * 5,
	}

	log.Println(`JSON-RPC server listening for HTTP traffic on ` + config.HttpBind)
	log.Fatalln(httpServer.ListenAndServe())

}

// TODO: replace RPC subsystem with something better.
type RpcHandler struct {
	project types.Project
}

var rpcEncoder = jsonEncoder{}

func (h RpcHandler) GetFiles(_ struct{}, res *[]string) error {
	out := make([]string, 0, len(h.project.Files))
	for path, _ := range h.project.Files {
		out = append(out, path)
	}
	*res = out
	return nil
}

func (h RpcHandler) GetContracts(file string, res *[]string) error {
	_file, ok := h.project.Files[file]
	if !ok {
		return fmt.Errorf(`file not found: %s`, file)
	}
	out := make([]string, 0, len(_file))
	for name, _ := range _file {
		out = append(out, name)
	}
	*res = out
	return nil
}

func (h RpcHandler) GetFile(req string, res *map[string]json.RawMessage) error {
	file, ok := h.project.Files[req]
	if !ok {
		return fmt.Errorf(`file not found: %s`, req)
	}
	out := make(map[string]json.RawMessage, len(file))
	for name, contract := range file {
		encoded, e := rpcEncoder.EncodeContract(contract)
		if e != nil {
			log.Panicln(e)
		}
		out[name] = encoded
	}
	*res = out
	return nil
}

type GetContractRequest struct {
	File     string `json:"file"`
	Contract string `json:"contract"`
}

func (h RpcHandler) GetContract(req GetContractRequest, res *json.RawMessage) error {

	file, ok := h.project.Files[req.File]
	if !ok {
		return fmt.Errorf(`file not found: %s`, req.File)
	}

	contract, ok := file[req.Contract]
	if !ok {
		return fmt.Errorf(`contract not found: %s`, req.Contract)
	}

	encoded, e := rpcEncoder.EncodeContract(contract)
	if e != nil {
		log.Panicln(e)
	}

	*res = json.RawMessage(encoded)
	return nil
}

type GetTypeRequest struct {
	File     string `json:"file"`
	Contract string `json:"contract"`
	Type     string `json:"type"`
}

func (h RpcHandler) GetType(req GetTypeRequest, res *json.RawMessage) error {

	file, ok := h.project.Files[req.File]
	if !ok {
		return fmt.Errorf(`file not found: %s`, req.File)
	}

	contract, ok := file[req.Contract]
	if !ok {
		return fmt.Errorf(`contract not found: %s`, req.Contract)
	}

	typ, ok := (types.Type)(nil), false

	for _, contract := range append([]*types.Contract{contract}, contract.Parents...) {
		if typ, ok = contract.Types[req.Type]; ok {
			break
		}
	}
	if !ok {
		return fmt.Errorf(`type not found: %s`, req.Type)
	}

	encoded, e := rpcEncoder.EncodeType(typ)
	if e != nil {
		log.Panicln(e)
	}

	*res = json.RawMessage(encoded)
	return nil
}

type GetOverloadsRequest struct {
	File     string `json:"file"`
	Contract string `json:"contract"`
	Function string `json:"function"`
}

func (h RpcHandler) GetOverloads(req GetOverloadsRequest, res *[]json.RawMessage) error {

	file, ok := h.project.Files[req.File]
	if !ok {
		return fmt.Errorf(`file not found: %s`, req.File)
	}

	contract, ok := file[req.Contract]
	if !ok {
		return fmt.Errorf(`contract not found: %s`, req.Contract)
	}

	sigs := make(map[string]struct{}, 8)
	out := make([]json.RawMessage, 0, len(contract.API))

	for _, contract := range append([]*types.Contract{contract}, contract.Parents...) {
		for _, function := range contract.API {
			if function.Name == req.Function {
				sig := string(function.SoliditySignature())
				if _, ok := sigs[sig]; ok {
					continue
				}
				encoded, e := rpcEncoder.EncodeFunction(function)
				if e != nil {
					log.Panicln(e)
				}
				out = append(out, encoded)
				sigs[sig] = struct{}{}
			}
		}
	}

	*res = out
	return nil
}

type GetFunctionRequest struct {
	File      string `json:"file"`
	Contract  string `json:"contract"`
	Signature string `json:"signature"`
}

func (h RpcHandler) GetFunction(req GetFunctionRequest, res *json.RawMessage) error {

	function, e := h.functionBySignature(req.File, req.Contract, req.Signature)
	if e != nil {
		return e
	}

	encoded, e := rpcEncoder.EncodeFunction(function)
	if e != nil {
		log.Panicln(e)
	}

	*res = json.RawMessage(encoded)
	return nil
}

type AuthenticationRequest struct {
	Authenticator string          `json:"authenticator"`
	Credentials   json.RawMessage `json:"credentials"`
}

func (h RpcHandler) Authenticate(req AuthenticationRequest, res *json.RawMessage) error {
	token, e := auth.Authenticate(req.Authenticator, req.Credentials)
	if e != nil {
		return e
	}
	*res = token
	return nil
}

type EncodeFunctionCallRequest struct {
	File      string          `json:"file"`
	Contract  string          `json:"contract"`
	Signature string          `json:"signature"`
	Arguments json.RawMessage `json:"arguments"`
}

type BinaryJSON []byte

func (j BinaryJSON) MarshalJSON() ([]byte, error) {
	s := `0x` + hex.EncodeToString([]byte(j))
	return json.Marshal(s)
}

func (h RpcHandler) EncodeFunctionCall(req EncodeFunctionCallRequest, res *BinaryJSON) error {
	function, e := h.functionBySignature(req.File, req.Contract, req.Signature)
	if e != nil {
		return e
	}
	calldata, e := abi.Encode(types.Tuple(function.Inputs), req.Arguments)
	if e != nil {
		return e
	}
	*res = append(keccak(function.SoliditySignature())[:4], calldata...)
	return nil
}

type RequestAuth struct {
	Provider string          `json:"authenticator"`
	Token    json.RawMessage `json:"token"`
}

type TransactionReceipt struct {
	Status            string                  `json:"status"`         // only set when tx done
	Root              string                  `json:"root,omitempty"` // only set while tx pending
	TransactionHash   string                  `json:"transactionHash"`
	BlockHash         string                  `json:"blockHash"`
	TransactionIndex  string                  `json:"transactionIndex"`
	BlockNumber       string                  `json:"blockNumber"`
	CumulativeGasUsed string                  `json:"cumulativeGasUsed"`
	GasUsed           string                  `json:"gasUsed"`
	ContractAddress   string                  `json:"contractAddress"`
	Logs              []TransactionReceiptLog `json:"logs"`
}

type TransactionReceiptLog struct {
	Removed          bool     `json:"removed"`
	LogIndex         string   `json:"logIndex"`
	TransactionIndex string   `json:"transactionIndex"`
	TransactionHash  string   `json:"transactionHash"`
	BlockHash        string   `json:"blockHash"`
	BlockNumber      string   `json:"blockNumber"`
	Address          string   `json:"address"`
	Data             string   `json:"data"`
	Topics           []string `json:"topics"`
}

type FunctionDispatchMode string

const (
	FunctionDispatchModeDefault         FunctionDispatchMode = `default`
	FunctionDispatchModeTransactionOnly FunctionDispatchMode = `transactionOnly`
	FunctionDispatchModeCallOnly        FunctionDispatchMode = `callOnly`
)

type DispatchFunctionCallRequest struct {
	EncodeFunctionCallRequest
	Target   string               `json:"target"`
	Value    json.Number          `json:"value"`
	GasPrice json.Number          `json:"gasPrice"`
	GasLimit json.Number          `json:"gasLimit"`
	Mode     FunctionDispatchMode `json:"mode"`
	Auth     RequestAuth          `json:"auth"`
}

type DispatchFunctionCallResponse struct {
	Result  json.RawMessage     `json:"result,omitempty"`
	Receipt *TransactionReceipt `json:"receipt,omitempty"`
}

func (h RpcHandler) DispatchFunctionCall(req DispatchFunctionCallRequest, res *DispatchFunctionCallResponse) error {

	if req.Value == "" {
		req.Value = "0"
	}

	if req.Mode == "" {
		req.Mode = FunctionDispatchModeDefault
	} else {
		if req.Mode != FunctionDispatchModeDefault &&
			req.Mode != FunctionDispatchModeTransactionOnly &&
			req.Mode != FunctionDispatchModeCallOnly {
			return fmt.Errorf(`invalid mode, available: default, transactionOnly, callOnly`)
		}
	}

	if req.Target == "" {
		return fmt.Errorf(`missing transaction target in request`)
	}

	gasLimit, gasPrice := uint64(defaultGasLimit), (*big.Int)(nil)

	if req.GasPrice == "" {
		gp := ""
		if e := EthClient.Call(&gp, `eth_gasPrice`); e != nil {
			return e
		}
		gasPrice, _ = new(big.Int).SetString(strip0xPrefix(gp), 16)
	} else {
		gp, ok := new(big.Int).SetString(string(req.GasPrice), 10)
		if !ok {
			return fmt.Errorf(`invalid gasPrice`)
		}
		gasPrice = gp
	}

	if req.GasLimit != "" {
		gl, e := strconv.ParseUint(string(req.GasLimit), 10, 64)
		if e != nil {
			return fmt.Errorf(`invalid gasLimit`)
		}
		gasLimit = gl
	}

	value, ok := new(big.Int).SetString(string(req.Value), 10)
	if !ok {
		return fmt.Errorf(`invalid value`)
	}

	function, e := h.functionBySignature(req.File, req.Contract, req.Signature)
	if e != nil {
		return e
	}

	calldata, e := abi.Encode(types.Tuple(function.Inputs), req.Arguments)
	if e != nil {
		return fmt.Errorf(`argument encoding error: %s`, e)
	}
	calldata = append(keccak(function.SoliditySignature())[:4], calldata...)

	key, e := auth.ExchangeToken(req.Auth.Provider, req.Auth.Token)
	if e != nil {
		return e // TODO: better errors
	}

	target := common.HexToAddress(req.Target)

	call := struct {
		From     string `json:"from"`
		To       string `json:"to"`
		Gas      string `json:"gas"`
		GasPrice string `json:"gasPrice"`
		Value    string `json:"value"`
		Data     string `json:"data"`
	}{
		From:     ensure0xPrefix(key.Address.String()),
		To:       ensure0xPrefix(target.String()),
		Gas:      ensure0xPrefix(strconv.FormatUint(gasLimit, 16)),
		GasPrice: ensure0xPrefix(gasPrice.Text(16)),
		Value:    ensure0xPrefix(value.Text(16)),
		Data:     ensure0xPrefix(hex.EncodeToString(calldata)),
	}

	// pure and view functions can be called without transacting
	if req.Mode == FunctionDispatchModeCallOnly ||
		(req.Mode == FunctionDispatchModeDefault &&
			(function.StateMutability == ast.StateMutabilityPure ||
				function.StateMutability == ast.StateMutabilityView)) {
		result := ""
		if e := EthClient.Call(&result, `eth_call`, call, `latest`); e != nil {
			return e // TODO: better error
		}
		if result == `0x` && len(function.Outputs) > 0 {
			return fmt.Errorf(`function call reverted -- gasLimit (%d) too low?`, gasLimit)
		}
		if len(function.Outputs) == 0 {
			*res = DispatchFunctionCallResponse{}
			return nil
		}
		code, e := hex.DecodeString(strip0xPrefix(result))
		if e != nil {
			return e // TODO: better error
		}
		decoded, e := abi.Decode(types.Tuple(function.Outputs), code)
		if e != nil {
			return e // TODO: context in error
		}
		*res = DispatchFunctionCallResponse{Result: decoded}
		return nil
	}

	nonce := uint64(0)
	{
		nc := ""
		if e := EthClient.Call(&nc, `eth_getTransactionCount`, key.Address, `pending`); e != nil {
			return fmt.Errorf(`failed to get nonce: %s`, e.Error())
		}
		nonce, _ = strconv.ParseUint(strip0xPrefix(nc), 16, 64)
	}

	transaction, e := ethtypes.SignTx(ethtypes.NewTransaction(nonce, target, value, gasLimit, gasPrice, calldata), signer, key.PrivateKey)
	if e != nil {
		return fmt.Errorf(`error signing transaction: %s`, e.Error())
	}

	data, e := rlp.EncodeToBytes(transaction)
	if e != nil {
		return e // TODO: better error
	}
	if e := EthClient.Call(nil, `eth_sendRawTransaction`, ensure0xPrefix(hex.EncodeToString(data))); e != nil {
		return e // TODO: better error
	}

	receipt := TransactionReceipt{Status: `pending`} // "pending" is placeholder

	for {
		if e := EthClient.Call(&receipt, `eth_getTransactionReceipt`, transaction.Hash()); e != nil {
			return e // TODO: better error
		}
		if receipt.Status == `pending` {
			time.Sleep(time.Second / 2)
			continue
		}
		break
	}

	if receipt.Status != `0x1` {
		return fmt.Errorf(`transaction reverted -- gasLimit (%d) too low?`, gasLimit)
	}

	if req.Mode == FunctionDispatchModeTransactionOnly {
		*res = DispatchFunctionCallResponse{Receipt: &receipt}
		return nil
	}

	// prevBlockNr := (receipt.BlockNumber - 1)
	blockNr, _ := new(big.Int).SetString(strip0xPrefix(receipt.BlockNumber), 16)
	prevBlockNr := new(big.Int).Sub(blockNr, big.NewInt(1))

	result := ""
	if e := EthClient.Call(&result, `eth_call`, call, ensure0xPrefix(prevBlockNr.Text(16))); e != nil {
		return e // TODO: better error
	}
	if result == `0x` && len(function.Outputs) > 0 {
		// TODO: transaction succeeded but call didn't... better response?
		*res = DispatchFunctionCallResponse{Receipt: &receipt}
		return nil
	}
	if len(function.Outputs) == 0 {
		*res = DispatchFunctionCallResponse{Receipt: &receipt}
		return nil
	}
	code, e := hex.DecodeString(strip0xPrefix(result))
	if e != nil {
		return e // TODO: better error
	}
	decoded, e := abi.Decode(types.Tuple(function.Outputs), code)
	if e != nil {
		return e // TODO: context in error
	}
	*res = DispatchFunctionCallResponse{Result: decoded, Receipt: &receipt}
	return nil

}

type CreateContractRequest struct {
	GetContractRequest
	Value    json.Number `json:"value"`
	GasPrice json.Number `json:"gasPrice"`
	GasLimit json.Number `json:"gasLimit"`
	Auth     RequestAuth `json:"auth"`
}

func (h RpcHandler) CreateContract(req CreateContractRequest, res *TransactionReceipt) error {

	file, ok := h.project.Files[req.File]
	if !ok {
		return fmt.Errorf(`file not found: %s`, req.File)
	}

	contract, ok := file[req.Contract]
	if !ok {
		return fmt.Errorf(`contract not found: %s`, req.Contract)
	}

	if req.Value == "" {
		req.Value = "0"
	}

	gasLimit, gasPrice := uint64(defaultGasLimit), (*big.Int)(nil)

	if req.GasPrice == "" {
		gp := ""
		if e := EthClient.Call(&gp, `eth_gasPrice`); e != nil {
			return e
		}
		gasPrice, _ = new(big.Int).SetString(strip0xPrefix(gp), 16)
	} else {
		gp, ok := new(big.Int).SetString(string(req.GasPrice), 10)
		if !ok {
			return fmt.Errorf(`invalid gasPrice`)
		}
		gasPrice = gp
	}

	if req.GasLimit != "" {
		gl, e := strconv.ParseUint(string(req.GasLimit), 10, 64)
		if e != nil {
			return fmt.Errorf(`invalid gasLimit`)
		}
		gasLimit = gl
	}

	value, ok := new(big.Int).SetString(string(req.Value), 10)
	if !ok {
		return fmt.Errorf(`invalid value`)
	}

	key, e := auth.ExchangeToken(req.Auth.Provider, req.Auth.Token)
	if e != nil {
		return e
	}

	nonce := uint64(0)
	{
		nc := ""
		if e := EthClient.Call(&nc, `eth_getTransactionCount`, key.Address, `pending`); e != nil {
			return fmt.Errorf(`failed to get nonce: %s`, e.Error())
		}
		nonce, _ = strconv.ParseUint(strip0xPrefix(nc), 16, 64)
	}

	transaction, e := ethtypes.SignTx(ethtypes.NewContractCreation(nonce, value, gasLimit, gasPrice, contract.Binary), signer, key.PrivateKey)
	if e != nil {
		return fmt.Errorf(`error signing transaction: %s`, e.Error())
	}

	{
		bs, e := rlp.EncodeToBytes(transaction)
		if e != nil {
			return e // TODO: better error
		}
		if e := EthClient.Call(nil, `eth_sendRawTransaction`, ensure0xPrefix(hex.EncodeToString(bs))); e != nil {
			return e // TODO: better error
		}
	}

	// TODO: cancel transactions pending for longer than a certain amount of time (gasPrice too low)
	// NOTE: failed transaction creations still result in a contract address but with no code in it

	for {

		receipt := TransactionReceipt{Status: `pending`}
		if e := EthClient.Call(&receipt, `eth_getTransactionReceipt`, transaction.Hash()); e != nil {
			return e // TODO: better error
		}
		if receipt.Status == `pending` {
			time.Sleep(time.Second / 2)
			continue
		}
		if receipt.Status != `0x1` { // 0x1 = success
			return fmt.Errorf(`contract creation reverted -- gasLimit (%d) too low?`, gasLimit)
		}
		*res = receipt
		break
	}

	return nil
}

func (h RpcHandler) functionBySignature(file, contract, signature string) (types.Function, error) {

	function := types.Function{}

	_file, ok := h.project.Files[file]
	if !ok {
		return function, fmt.Errorf(`file not found: %s`, file)
	}

	_contract, ok := _file[contract]
	if !ok {
		return function, fmt.Errorf(`contract not found: %s`, contract)
	}

	sigs := make([]string, 0, 16)

	for _, contract := range append([]*types.Contract{_contract}, _contract.Parents...) {
		for _, function := range contract.API {
			sig := string(function.SoliditySignature())
			if sig == signature {
				return function, nil
			}
			sigs = append(sigs, sig)
		}
	}

	return function, fmt.Errorf(`function signature not found: %s. available are: %s`, signature, strings.Join(sigs, `, `))

}

type jsonEncoder struct{}

func (codec jsonEncoder) EncodeProject(project types.Project) ([]byte, error) {
	files := make(map[string]map[string]json.RawMessage)
	for path, contracts := range project.Files {
		files[path] = make(map[string]json.RawMessage, len(contracts))
		for name, contract := range contracts {
			encodedContract, e := codec.EncodeContract(contract)
			if e != nil {
				return nil, e
			}
			files[path][name] = json.RawMessage(encodedContract)
		}
	}
	return json.Marshal(struct {
		Kind  string                                `json:"kind"`
		Files map[string]map[string]json.RawMessage `json:"files"`
	}{
		Kind:  `project`,
		Files: files,
	})
}

func (codec jsonEncoder) EncodeContract(contract *types.Contract) ([]byte, error) {
	parents := make([]string, len(contract.Parents), len(contract.Parents))
	for i, parent := range contract.Parents {
		parents[i] = (parent.File + ":" + parent.Name)
	}
	api := make(map[string]json.RawMessage, len(contract.API))
	for signature, function := range contract.API {
		encoded, e := codec.EncodeFunction(function)
		if e != nil {
			return nil, e
		}
		api[signature] = encoded
	}
	types := make(map[string]json.RawMessage, len(contract.Types))
	for name, typ := range contract.Types {
		encoded, e := codec.EncodeType(typ)
		if e != nil {
			return nil, e
		}
		types[name] = encoded
	}
	return json.Marshal(struct {
		Kind         string                     `json:"kind"`
		File         string                     `json:"file"`
		Name         string                     `json:"name"`
		Parents      []string                   `json:"parents"`
		NatSpec      string                     `json:"natSpec"`
		ContractKind ast.ContractKind           `json:"contractKind"`
		API          map[string]json.RawMessage `json:"api"`
		Types        map[string]json.RawMessage `json:"types"`
		Binary       BinaryJSON                 `json:"binary"`
	}{
		Kind:         `contract`,
		File:         contract.File,
		Name:         contract.Name,
		Parents:      parents,
		NatSpec:      contract.NatSpec,
		ContractKind: contract.Kind,
		API:          api,
		Types:        types,
		Binary:       BinaryJSON(contract.Binary),
	})
}

func (codec jsonEncoder) EncodeFunction(function types.Function) ([]byte, error) {
	inputs := make([]json.RawMessage, len(function.Inputs), len(function.Inputs))
	for i, input := range function.Inputs {
		encodedInput, e := codec.EncodeType(input)
		if e != nil {
			return nil, e
		}
		inputs[i] = encodedInput
	}
	outputs := make([]json.RawMessage, len(function.Outputs), len(function.Outputs))
	for i, output := range function.Outputs {
		encodedOutput, e := codec.EncodeType(output)
		if e != nil {
			return nil, e
		}
		outputs[i] = encodedOutput
	}
	sig := function.SoliditySignature()
	return json.Marshal(struct {
		Kind        string            `json:"kind"`
		Signature   string            `json:"signature"`
		Fingerprint string            `json:"fingerprint"`
		Name        string            `json:"name"`
		NatSpec     string            `json:"natSpec"`
		Visibility  ast.Visibility    `json:"visibility"`
		Inputs      []json.RawMessage `json:"inputs"`
		Outputs     []json.RawMessage `json:"outputs"`
	}{
		Kind:        `function`,
		Signature:   string(sig),
		Fingerprint: hex.EncodeToString(keccak(sig)[:4]),
		Name:        function.Name,
		NatSpec:     function.NatSpec,
		Visibility:  function.Visibility,
		Inputs:      inputs,
		Outputs:     outputs,
	})
}

func (codec jsonEncoder) EncodeType(typ types.Type) ([]byte, error) {
	definition, e := codec.encodeType(typ)
	if e != nil {
		return nil, e
	}
	return json.Marshal(struct {
		Kind       string          `json:"kind"`
		Definition json.RawMessage `json:"definition"`
	}{
		Kind:       `type`,
		Definition: definition,
	})
}

func (codec jsonEncoder) encodeType(typ types.Type) ([]byte, error) {
	switch t := typ.(type) {

	case types.Event:
		args := make([]json.RawMessage, len(t.Args), len(t.Args))
		for i, typ := range t.Args {
			arg, e := codec.encodeType(typ)
			if e != nil {
				return nil, e
			}
			args[i] = arg
		}
		return json.Marshal(struct {
			Kind string            `json:"kind"`
			Name string            `json:"name"`
			Args []json.RawMessage `json:"args"`
		}{
			Kind: `event`,
			Name: string(t.Name),
			Args: args,
		})

	case types.Tuple:
		types := make([]json.RawMessage, len(t), len(t))
		for i, typ := range t {
			arg, e := codec.encodeType(typ)
			if e != nil {
				return nil, e
			}
			types[i] = arg
		}
		return json.Marshal(struct {
			Kind  string            `json:"kind"`
			Types []json.RawMessage `json:"types"`
		}{
			Kind:  `tuple`,
			Types: types,
		})

	case types.Elementary:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		}{
			Kind: `elementary`,
			Name: string(t),
		})
	case types.Struct:
		types := make([]json.RawMessage, len(t.Types), len(t.Types))
		for i, subType := range t.Types {
			encoded, e := codec.encodeType(subType)
			if e != nil {
				return nil, e
			}
			types[i] = encoded
		}
		return json.Marshal(struct {
			Kind  string            `json:"kind"`
			Keys  []string          `json:"keys"`
			Types []json.RawMessage `json:"types"`
		}{
			Kind:  `struct`,
			Keys:  t.Keys,
			Types: types,
		})
	case types.Array:
		subType, e := codec.encodeType(t.Type)
		if e != nil {
			return nil, e
		}
		return json.Marshal(struct {
			Kind   string          `json:"kind"`
			Length int             `json:"length"`
			Type   json.RawMessage `json:"type"`
		}{
			Kind:   `array`,
			Length: t.Length,
			Type:   subType,
		})
	case types.Mapping:
		key, e := codec.encodeType(t.Key)
		if e != nil {
			return nil, e
		}
		value, e := codec.encodeType(t.Value)
		if e != nil {
			return nil, e
		}
		return json.Marshal(struct {
			Kind  string          `json:"kind"`
			Key   json.RawMessage `json:"key"`
			Value json.RawMessage `json:"value"`
		}{
			Kind:  `mapping`,
			Key:   key,
			Value: value,
		})
	case types.Enum:
		return json.Marshal(struct {
			Kind   string   `json:"kind"`
			Values []string `json:"values"`
		}{
			Kind:   `enum`,
			Values: []string(t),
		})
	case types.Named:
		encoded, e := codec.encodeType(t.Type)
		if e != nil {
			return nil, e
		}
		return json.Marshal(struct {
			Kind string          `json:"kind"`
			Name string          `json:"name"`
			Type json.RawMessage `json:"type"`
		}{
			Kind: `named`,
			Name: t.Name,
			Type: encoded,
		})
	case types.ContractAddress:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		}{
			Kind: `contractAddress`,
			Name: string(t),
		})
	case types.InterfaceAddress:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		}{
			Kind: `interfaceAddress`,
			Name: string(t),
		})
	case types.LibraryAddress:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		}{
			Kind: `libraryAddress`,
			Name: string(t),
		})
	}
	log.Panicf(`unexpected type in jsonEncoder.encodeType: %T`, typ)
	return nil, nil // shut up compiler
}

func keccak(input []byte) []byte {
	hash := sha3.NewKeccak256()
	if n, e := hash.Write(input); n != len(input) || e != nil {
		log.Fatalln(e)
	}
	_ = hex.EncodeToString
	return hash.Sum(nil)
}

func ensure0xPrefix(s string) string {
	if len(s) < 2 || s[:2] != `0x` {
		return `0x` + s
	}
	return s
}

func strip0xPrefix(s string) string {
	if len(s) > 2 && s[:2] == `0x` {
		return s[2:]
	}
	return s
}
