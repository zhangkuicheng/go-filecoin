package main

import (
	"context"
	"math/big"

	hamt "github.com/ipfs/go-hamt-ipld"
)

var InitialNetworkTokens = big.NewInt(2000000000)

func CreateGenesisBlock(cs *hamt.CborIpldStore) (*Block, error) {
	ctx := context.Background()

	stateRoot := hamt.NewNode(cs)
	s := &State{root: stateRoot, store: cs}

	genesis := new(Block)

	// Set up filecoin token contract
	tokenState := s.NewContractState()
	if err := tokenState.Set(ctx, string(FilecoinContractAddr), InitialNetworkTokens.Bytes()); err != nil {
		return nil, err
	}

	tsCid, err := cs.Put(ctx, tokenState.n)
	if err != nil {
		return nil, err
	}

	filTokActor := &Actor{
		Code:   FilecoinContractAddr,
		Memory: tsCid,
	}
	if err := s.SetActor(ctx, FilecoinContractAddr, filTokActor); err != nil {
		return nil, err
	}

	// Set up storage miner contract
	storageState := s.NewContractState()
	stsCid, err := cs.Put(ctx, storageState.n)
	if err != nil {
		return nil, err
	}

	storMarketActor := &Actor{
		Code:   StorageContractCodeAddress,
		Memory: stsCid,
	}
	if err := s.SetActor(ctx, StorageContractAddress, storMarketActor); err != nil {
		return nil, err
	}

	srcid, err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	genesis.StateRoot = srcid
	return genesis, nil
}
