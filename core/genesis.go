package core

import (
	"context"
	"math/big"

	contract "github.com/filecoin-project/playground/go-filecoin/contract"
	types "github.com/filecoin-project/playground/go-filecoin/types"
	hamt "gx/ipfs/QmeEgzPRAjisT3ndLSR8jrrZAZyWd3nx2mpZU4S7mCQzYi/go-hamt-ipld"
)

var InitialNetworkTokens = big.NewInt(2000000000)

func CreateGenesisBlock(cs *hamt.CborIpldStore) (*types.Block, error) {
	ctx := context.Background()

	stateRoot := hamt.NewNode(cs)
	s := contract.NewState(cs, stateRoot)

	genesis := new(types.Block)

	// Set up filecoin token contract
	tokenState := s.NewContractState()
	if err := tokenState.Set(ctx, string(contract.FilecoinContractAddr), InitialNetworkTokens.Bytes()); err != nil {
		return nil, err
	}

	tsCid, err := cs.Put(ctx, tokenState.Node())
	if err != nil {
		return nil, err
	}

	filTokActor := &contract.Actor{
		Code:   contract.FilecoinContractCid,
		Memory: tsCid,
	}
	if err := s.SetActor(ctx, contract.FilecoinContractAddr, filTokActor); err != nil {
		return nil, err
	}

	// Set up storage miner contract
	storageState := s.NewContractState()
	stsCid, err := cs.Put(ctx, storageState.Node())
	if err != nil {
		return nil, err
	}

	storMarketActor := &contract.Actor{
		Code:   contract.StorageContractCodeCid,
		Memory: stsCid,
	}
	if err := s.SetActor(ctx, contract.StorageContractAddress, storMarketActor); err != nil {
		return nil, err
	}

	srcid, err := s.Flush(ctx)
	if err != nil {
		return nil, err
	}

	genesis.StateRoot = srcid
	return genesis, nil
}
