package core

import (
	"context"
	"math/big"
)

var MinerContractCodeHash = identCid("fcminer")

type MinerContract struct {
	Owner         Address
	Pledge        *big.Int
	Power         *big.Int
	LockedStorage *big.Int

	s *ContractState
}

// LoadState is purely a helper function that loads information from the
// contract state into the structs variables.
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

// Flush writes the values in the structs fields to the state tree
func (mc *MinerContract) Flush(ctx context.Context) error {
	if err := mc.s.Set(ctx, "owner", []byte(mc.Owner)); err != nil {
		return err
	}

	if err := mc.s.Set(ctx, "pledge", mc.Pledge.Bytes()); err != nil {
		return err
	}

	if err := mc.s.Set(ctx, "power", mc.Power.Bytes()); err != nil {
		return err
	}

	if err := mc.s.Set(ctx, "locked", mc.LockedStorage.Bytes()); err != nil {
		return err
	}

	return nil
}
