package main

import (
	"context"
	"fmt"
	"math/big"

	// TODO: no usage of this package directly
	hamt "github.com/ipfs/go-hamt-ipld"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

var FilecoinContractCid = identCid("filecoin")
var FilecoinContractAddr = Address("filecoin")

func identCid(s string) *cid.Cid {
	h, err := mh.Sum([]byte(s), mh.ID, len(s))
	if err != nil {
		panic(err)
	}
	return cid.NewCidV1(cid.Raw, h)
}

// CallContext is information accessible to the contract during a given invocation
type CallContext struct {
	Ctx           context.Context
	From          Address
	State         *State
	ContractState *ContractState
}

type Contract interface {
	Call(ctx *CallContext, method string, args []interface{}) (interface{}, error)
}

type FilecoinTokenContract struct{}

var ErrMethodNotFound = fmt.Errorf("unrecognized method")

func (ftc *FilecoinTokenContract) Call(ctx *CallContext, method string, args []interface{}) (interface{}, error) {
	switch method {
	case "transfer":
		return ftc.transfer(ctx, args)
	case "getBalance":
		return ftc.getBalance(ctx, args)
	default:
		return nil, ErrMethodNotFound
	}
}

func (ftc *FilecoinTokenContract) getBalance(ctx *CallContext, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("getBalance takes exactly 1 argument")
	}

	addr, ok := args[0].(Address)
	if !ok {
		return nil, fmt.Errorf("argument must be an Address")
	}

	cs := ctx.ContractState
	accData, err := cs.Get(ctx.Ctx, string(addr))
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(accData), nil
}

type Account struct {
	Balance *big.Int
	Nonce   uint64
}

func addressCast(i interface{}) (Address, error) {
	switch i := i.(type) {
	case Address:
		return i, nil
	case string:
		if i[:2] == "0x" {
			return ParseAddress(i)
		}
		return Address(i), nil
	default:
		return "", fmt.Errorf("first argument must be an Address")
	}
}

// very temporary hack
func numberCast(i interface{}) (*big.Int, error) {
	switch i := i.(type) {
	case string:
		n, ok := big.NewInt(0).SetString(i, 10)
		if !ok {
			return nil, fmt.Errorf("arg must be a number")
		}
		return n, nil
	case *big.Int:
		return i, nil
	case float64:
		return big.NewInt(int64(i)), nil
	case []byte:
		return big.NewInt(0).SetBytes(i), nil
	default:
		fmt.Printf("type is: %T\n", i)
		panic("noo")
	}
}

func (ftc *FilecoinTokenContract) transfer(ctx *CallContext, args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("transfer takes exactly 2 arguments")
	}

	toAddr, err := addressCast(args[0])
	if err != nil {
		return nil, err
	}

	amount, err := numberCast(args[1])
	if err != nil {
		return nil, err
	}

	cs := ctx.ContractState
	fromData, err := cs.Get(ctx.Ctx, string(ctx.From))
	if err != nil && err != hamt.ErrNotFound {
		return nil, err
	}

	fromBalance := big.NewInt(0).SetBytes(fromData)

	if fromBalance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("not enough funds")
	}

	fromBalance = fromBalance.Sub(fromBalance, amount)

	toData, err := cs.Get(ctx.Ctx, string(toAddr))
	if err != nil && err != hamt.ErrNotFound {
		return nil, err
	}

	toBalance := big.NewInt(0).SetBytes(toData)
	toBalance = toBalance.Add(toBalance, amount)

	if err := cs.Set(ctx.Ctx, string(ctx.From), fromBalance.Bytes()); err != nil {
		return nil, err
	}

	if err := cs.Set(ctx.Ctx, string(toAddr), toBalance.Bytes()); err != nil {
		return nil, err
	}

	return nil, nil
}
