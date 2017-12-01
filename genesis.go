package main

import (
	"context"
	"math/big"

	hamt "github.com/ipfs/go-hamt-ipld"
)

var InitialNetworkTokens = big.NewInt(2000000000)

func CreateGenesisBlock(cs *hamt.CborIpldStore) (*Block, error) {
	ctx := context.Background()
	genesis := new(Block)
	tokenState := hamt.NewNode(cs)
	if err := tokenState.Set(ctx, string(FilecoinContractAddr), InitialNetworkTokens.Bytes()); err != nil {
		return nil, err
	}

	tsCid, err := cs.Put(ctx, tokenState)
	if err != nil {
		return nil, err
	}

	stateRoot := hamt.NewNode(cs)
	s := &State{root: stateRoot, store: cs}

	filTokActor := &Actor{
		Code:   FilecoinContractAddr,
		Memory: tsCid,
	}
	if err := s.SetActor(ctx, FilecoinContractAddr, filTokActor); err != nil {
		return nil, err
	}

	srcid, err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	genesis.StateRoot = srcid
	return genesis, nil
}
