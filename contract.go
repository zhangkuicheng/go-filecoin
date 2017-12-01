package main

import (
	"context"
	"fmt"
	"math/big"

	hamt "github.com/ipfs/go-hamt-ipld"
	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
)

var FilecoinContractAddr = Address("filecoin")

type CallContext struct {
	Ctx  context.Context
	From Address
}

type Contract interface {
	Call(ctx *CallContext, method string, args []interface{}) (interface{}, error)
	LoadState(s *hamt.Node) error

	// TODO: this signature sucks, need to get the abstractions right
	Flush(ctx context.Context, cs *hamt.CborIpldStore) (*cid.Cid, error)
}

type FilecoinTokenContract struct {
	s *hamt.Node
}

func (ftc *FilecoinTokenContract) LoadState(s *hamt.Node) error {
	ftc.s = s
	return nil
}

func (ftc *FilecoinTokenContract) Call(ctx *CallContext, method string, args []interface{}) (interface{}, error) {
	switch method {
	case "transfer":
		return ftc.transfer(ctx, args)
	case "getBalance":
		return ftc.getBalance(ctx, args)
	default:
		return nil, fmt.Errorf("unrecognized method")
	}
}

func (ftc *FilecoinTokenContract) Flush(ctx context.Context, cs *hamt.CborIpldStore) (*cid.Cid, error) {
	if err := ftc.s.Flush(ctx); err != nil {
		return nil, err
	}

	return cs.Put(ctx, ftc.s)
}

func (ftc *FilecoinTokenContract) getBalance(ctx *CallContext, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("getBalance takes exactly 1 argument")
	}

	addr, ok := args[0].(Address)
	if !ok {
		return nil, fmt.Errorf("argument must be an Address")
	}

	accData, err := ftc.s.Find(ctx.Ctx, string(addr))
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(accData), nil
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

	fromData, err := ftc.s.Find(ctx.Ctx, string(ctx.From))
	if err != nil && err != hamt.ErrNotFound {
		return nil, err
	}

	fromBalance := big.NewInt(0).SetBytes(fromData)

	if fromBalance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("not enough funds")
	}

	fromBalance = fromBalance.Sub(fromBalance, amount)

	toData, err := ftc.s.Find(ctx.Ctx, string(toAddr))
	if err != nil && err != hamt.ErrNotFound {
		return nil, err
	}

	toBalance := big.NewInt(0).SetBytes(toData)
	toBalance = toBalance.Add(toBalance, amount)

	if err := ftc.s.Set(ctx.Ctx, string(ctx.From), fromBalance.Bytes()); err != nil {
		return nil, err
	}

	if err := ftc.s.Set(ctx.Ctx, string(toAddr), toBalance.Bytes()); err != nil {
		return nil, err
	}

	return nil, nil
}
