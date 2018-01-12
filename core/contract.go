package core

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	// TODO: no usage of this package directly
	hamt "github.com/ipfs/go-hamt-ipld"

	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
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
		return typedCall(ctx, args, ftc.getBalance)
	default:
		return nil, ErrMethodNotFound
	}
}

func (ftc *FilecoinTokenContract) getBalance(ctx *CallContext, addr Address) (interface{}, error) {
	fmt.Println("getting address: ", addr)

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

var (
	addrType = reflect.TypeOf(Address(""))
)

func typedCall(cctx *CallContext, args []interface{}, f interface{}) (interface{}, error) {
	fval := reflect.ValueOf(f)
	if fval.Kind() != reflect.Func {
		return nil, fmt.Errorf("must pass a function")
	}

	ftype := fval.Type()
	if ftype.In(0) != reflect.TypeOf(&CallContext{}) {
		return nil, fmt.Errorf("first parameter must be call context")
	}

	var callargs []reflect.Value
	for i := 1; i < ftype.NumIn(); i++ {
		switch ftype.In(i) {
		case addrType:
			v, err := castToAddress(reflect.ValueOf(args[i-1]))
			if err != nil {
				return nil, err
			}
			callargs = append(callargs, v)
		default:
			return nil, fmt.Errorf("unsupported type: %s", ftype.In(i))
		}
	}

	out := fval.Call(callargs)
	return out[0].Interface(), out[1].Interface().(error)
}

func castToAddress(v reflect.Value) (reflect.Value, error) {
	return v, nil
}
