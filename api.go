package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	path "github.com/ipfs/go-ipfs/path"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
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
		"addAsk":      addAsk,
		"addBid":      addBid,
		"getOrders":   getOpenOrders,
		"makeDeal":    makeDeal,
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
		From:   fcn.Addresses[0],
		To:     FilecoinContractAddr,
		Method: "transfer",
		Params: []interface{}{toaddr, amount},
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
		From:   fcn.Addresses[0],
		To:     StorageContractAddress,
		Method: "createMiner",
		Params: []interface{}{pledge},
	}

	return nil, fcn.SendNewTransaction(tx)

	// TODO: need to wait for output
}

func addBid(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	if len(rpc.Args) != 2 {
		return nil, fmt.Errorf("must pass two arguments")
	}

	fmt.Println(rpc.Args)
	price, err := strconv.Atoi(rpc.Args[0])
	if err != nil {
		return nil, err
	}

	size, err := strconv.Atoi(rpc.Args[1])
	if err != nil {
		return nil, err
	}

	b := &Bid{
		Size:  uint64(size),
		Price: big.NewInt(int64(price)),
	}

	tx := &Transaction{
		From:   fcn.Addresses[0],
		To:     StorageContractAddress,
		Method: "addBid",
		Params: []interface{}{b},
	}

	return nil, fcn.SendNewTransaction(tx)
}

func addAsk(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	if len(rpc.Args) != 3 {
		fmt.Println(rpc.Args)
		return nil, fmt.Errorf("must pass three arguments")
	}

	fmt.Println(rpc.Args)
	miner, err := ParseAddress(rpc.Args[0])
	if err != nil {
		return nil, err
	}

	price, err := strconv.Atoi(rpc.Args[1])
	if err != nil {
		return nil, err
	}

	size, err := strconv.Atoi(rpc.Args[2])
	if err != nil {
		return nil, err
	}

	a := &Ask{
		Size:  uint64(size),
		Price: big.NewInt(int64(price)),
	}

	tx := &Transaction{
		From:   fcn.Addresses[0],
		To:     StorageContractAddress,
		Method: "addAsk",
		Params: []interface{}{miner, a},
	}

	return nil, fcn.SendNewTransaction(tx)
}

func getOpenOrders(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	ctx := context.Background()
	sact, err := fcn.stateRoot.GetActor(ctx, StorageContractAddress)
	if err != nil {
		return nil, err
	}

	c, err := sact.LoadContract(ctx, fcn.stateRoot)
	if err != nil {
		return nil, err
	}

	sc, ok := c.(*StorageContract)
	if !ok {
		return nil, fmt.Errorf("was not actually a storage contract somehow")
	}

	switch rpc.Args[0] {
	case "bids":
		ids, err := sc.loadArray(ctx, bidsArrKey)
		if err != nil {
			return nil, err
		}
		var bids []*Bid
		for _, id := range ids {
			data, err := sc.st.Get(ctx, "b"+fmt.Sprint(id))
			if err != nil {
				return nil, err
			}
			var b Bid
			if err := json.Unmarshal(data, &b); err != nil {
				return nil, err
			}
			bids = append(bids, &b)
		}
		return bids, nil
	case "asks":
		ids, err := sc.loadArray(ctx, asksArrKey)
		if err != nil {
			return nil, err
		}
		var asks []*Ask
		for _, id := range ids {
			data, err := sc.st.Get(ctx, "a"+fmt.Sprint(id))
			if err != nil {
				return nil, err
			}
			var a Ask
			if err := json.Unmarshal(data, &a); err != nil {
				return nil, err
			}
			asks = append(asks, &a)
		}
		return asks, nil
	default:
		return nil, fmt.Errorf("argument must be either 'bids' or 'asks'")
	}
}

func makeDeal(fcn *FilecoinNode, rpc *RPC) (interface{}, error) {
	askid, err := strconv.Atoi(rpc.Args[0])
	if err != nil {
		return nil, err
	}

	bidid, err := strconv.Atoi(rpc.Args[1])
	if err != nil {
		return nil, err
	}

	dataref, err := cid.Decode(rpc.Args[2])
	if err != nil {
		return nil, err
	}

	// verify we have dataref locally

	d := &Deal{
		Ask:     uint64(askid),
		Bid:     uint64(bidid),
		DataRef: dataref,
	}

	// Send deal to miner. Miner responds 'accept'

	Boat := struct {
		Floaty  bool
		Anchor  string
		Captain Address
		Cargo   *Deal
	}{}

	Boat.Cargo = d

	// dig says its fine now

	return nil, err
}
