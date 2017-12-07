package main

import (
	"context"
	"math/big"

	// TODO: no usage of this package directly
	hamt "github.com/ipfs/go-hamt-ipld"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
)

var MinerContractCodeHash = identCid("fcminer")

type MinerContract struct {
	Owner         Address
	Pledge        *big.Int
	Power         *big.Int
	LockedStorage *big.Int

	s *ContractState
}

func (mc *MinerContract) LoadState(s *ContractState) error {
	ownb, err := s.Get(context.TODO(), "owner")
	if err != nil {
		return err
	}
	mc.Owner = Address(ownb)

	plb, err := s.Get(context.TODO(), "pledge")
	if err != nil {
		return err
	}
	mc.Pledge = big.NewInt(0).SetBytes(plb)

	powb, err := s.Get(context.TODO(), "power")
	if err != nil {
		return err
	}
	mc.Power = big.NewInt(0).SetBytes(powb)

	lckb, err := s.Get(context.TODO(), "locked")
	if err != nil {
		return err
	}
	mc.LockedStorage = big.NewInt(0).SetBytes(lckb)

	mc.s = s

	return nil
}

func (mc *MinerContract) Call(ctx *CallContext, method string, args []interface{}) (interface{}, error) {
	panic("NYI")
}

func (mc *MinerContract) Flush(ctx context.Context, cs *hamt.CborIpldStore) (*cid.Cid, error) {
	if err := mc.s.Set(ctx, "owner", []byte(mc.Owner)); err != nil {
		return nil, err
	}

	if err := mc.s.Set(ctx, "pledge", mc.Pledge.Bytes()); err != nil {
		return nil, err
	}

	if err := mc.s.Set(ctx, "power", mc.Power.Bytes()); err != nil {
		return nil, err
	}

	if err := mc.s.Set(ctx, "locked", mc.LockedStorage.Bytes()); err != nil {
		return nil, err
	}

	if err := mc.s.Flush(ctx); err != nil {
		return nil, err
	}

	return cs.Put(ctx, mc.s.n) // bad abstractions...
}
