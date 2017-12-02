package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	path "github.com/ipfs/go-ipfs/path"
)

type BadApi struct {
	fcn      *FilecoinNode
	commands map[string]badRpcFunc
}

type badRpcFunc func(*FilecoinNode, *RPC) (interface{}, error)

func NewBadApi(fcn *FilecoinNode) *BadApi {
	ba := &BadApi{fcn: fcn}

	ba.commands = map[string]badRpcFunc{
		"listAddrs":   listAddrs,
		"newAddr":     newAddrCmd,
		"wantlist":    wantlistCmd,
		"dagGet":      dagGetCmd,
		"sendTx":      sendTxCmd,
		"getBalance":  getBalanceCmd,
		"createMiner": createMinerCmd,
	}

	return ba
}

type RPC struct {
	Method string
	Args   []string
}

func (ba *BadApi) ApiHandlerPleaseReplace(w http.ResponseWriter, r *http.Request) {
	// TODO: don't use a json rpc. it sucks. but its easy.
	var rpc RPC
	if err := json.NewDecoder(r.Body).Decode(&rpc); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	command, ok := ba.commands[rpc.Method]
	if !ok {
		w.WriteHeader(404)
		return
	}

	out, err := command(ba.fcn, &rpc)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(out)
}

func listAddrs(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	return fcn.Addresses, nil
}

func newAddrCmd(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	naddr := createNewAddress()
	fcn.Addresses = append(fcn.Addresses, naddr)
	return naddr, nil
}

func wantlistCmd(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	var cstrs []string
	for _, c := range fcn.bswap.GetWantlist() {
		cstrs = append(cstrs, c.String())
	}
	return strings.Join(cstrs, "\n"), nil
}

func dagGetCmd(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	ctx := context.Background()
	p, err := path.ParsePath(rpc.Args[0])
	if err != nil {
		return nil, err
	}

	res := path.NewBasicResolver(fcn.dag)
	nd, err := res.ResolvePath(ctx, p)
	if err != nil {
		return nil, err
	}

	b, err := json.MarshalIndent(nd, "", "  ")
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func sendTxCmd(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	if len(rpc.Args) != 2 {
		return nil, fmt.Errorf("must pass two arguments")
	}
	amount, ok := big.NewInt(0).SetString(rpc.Args[0], 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse amount")
	}
	toaddr, err := ParseAddress(rpc.Args[1])
	if err != nil {
		return nil, err
	}

	tx := &Transaction{
		FROMTEMP: fcn.Addresses[0],
		To:       FilecoinContractAddr,
		Method:   "transfer",
		Params:   []interface{}{toaddr, amount},
	}

	fcn.SendNewTransaction(tx)
	return nil, nil
}

func getBalanceCmd(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	ctx := context.Background()
	if len(rpc.Args) != 1 {
		return nil, fmt.Errorf("must pass address as argument")
	}

	addr, err := ParseAddress(rpc.Args[0])
	if err != nil {
		return nil, err
	}

	act, err := fcn.stateRoot.GetActor(ctx, FilecoinContractAddr)
	if err != nil {
		return nil, err
	}

	ct, err := act.LoadContract(ctx, fcn.stateRoot)
	if err != nil {
		return nil, err
	}

	cctx := &CallContext{Ctx: context.Background()}
	return ct.Call(cctx, "getBalance", []interface{}{addr})
}

func createMinerCmd(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	if len(rpc.Args) != 1 {
		return nil, fmt.Errorf("must pass pledge as argument")
	}

	pledge, ok := big.NewInt(0).SetString(rpc.Args[0], 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse pledge as number")
	}

	tx := &Transaction{
		FROMTEMP: fcn.Addresses[0],
		To:       StorageContractAddress,
		Method:   "createMiner",
		Params:   []interface{}{pledge},
	}

	return nil, fcn.SendNewTransaction(tx)
}
