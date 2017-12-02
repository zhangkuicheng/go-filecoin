package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"

	hamt "github.com/ipfs/go-hamt-ipld"
	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
)

var StorageContractCodeAddress = Address("storageContract")
var StorageContractAddress = Address("storageContractAddr")

// Only Duration or BlockHeight are allowed to be defined
type Bid struct {
	Expiry uint64

	Price *big.Int

	Size uint64

	// number of blocks, from the deal being commited to the chain
	Duration uint64

	// fixed block height
	BlockHeight uint64

	Collateral *big.Int

	//Coding      ErasureCoding
}

type Ask struct {
	Expiry uint64

	Price *big.Int

	Size uint64
}

type StorageContract struct {
	st *ContractState

	bidCount int64
	askCount int64
}

func (sc *StorageContract) LoadState(st *ContractState) error {
	sc.st = st
	bidsd, err := st.Get(context.Background(), "bids")
	if err != nil && err != hamt.ErrNotFound {
		return err
	}

	sc.bidCount = big.NewInt(0).SetBytes(bidsd).Int64()

	asksd, err := st.Get(context.Background(), "asks")
	if err != nil && err != hamt.ErrNotFound {
		return err
	}

	sc.askCount = big.NewInt(0).SetBytes(asksd).Int64()

	return nil
}

func (sc *StorageContract) Call(ctx *CallContext, method string, args []interface{}) (interface{}, error) {
	switch method {
	case "addBid":
		return sc.addBid(ctx, args[0])
	case "addAsk":
		return sc.addAsk(ctx, args[0])
	case "createMiner":
		return sc.createMiner(ctx, args)
	default:
		return nil, ErrMethodNotFound
	}
}

func (sc *StorageContract) Flush(ctx context.Context, cs *hamt.CborIpldStore) (*cid.Cid, error) {
	return cs.Put(ctx, sc.st.n)
}

func castBid(i interface{}) (*Bid, error) {
	fmt.Printf("bid: %#v\n", i)
	panic("halten sie!")
}

func (sc *StorageContract) addBid(ctx *CallContext, arg interface{}) (interface{}, error) {
	b, err := castBid(arg)
	if err != nil {
		return nil, err
	}

	if err := sc.validateBid(b); err != nil {
		return nil, err
	}

	bidID := sc.bidCount
	sc.bidCount++

	data, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}
	if err := sc.st.Set(ctx.Ctx, fmt.Sprint(bidID), data); err != nil {
		return nil, err
	}

	return bidID, nil
}

func (sc *StorageContract) validateBid(b *Bid) error {
	// check all the fields look good

	// need to check client has enough filecoin to lock up

	return nil
}

func (sc *StorageContract) addAsk(ctx *CallContext, arg interface{}) (interface{}, error) {
	return nil, nil
}

func (sc *StorageContract) createMiner(ctx *CallContext, args []interface{}) (interface{}, error) {
	pledge, err := numberCast(args[0])
	if err != nil {
		return nil, err
	}

	nminer := &MinerContract{
		Owner:         ctx.From,
		Pledge:        pledge,
		LockedStorage: big.NewInt(0),
		Power:         big.NewInt(0),
		s:             ctx.State.NewContractState(),
	}

	mem, err := nminer.Flush(ctx.Ctx, ctx.State.store)
	if err != nil {
		return nil, err
	}

	ca := compMinerContractAddress(nminer)

	act := &Actor{
		Code:   MinerContractCodeHash,
		Memory: mem,
	}
	if err := ctx.State.SetActor(ctx.Ctx, ca, act); err != nil {
		return nil, err
	}

	return ca, nil
}

func compMinerContractAddress(mc *MinerContract) Address {
	b, err := json.Marshal(mc)
	if err != nil {
		panic(err)
	}

	h := sha256.Sum256(b)
	return Address(h[:20])
}
