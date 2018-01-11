package core

import (
	"context"
	"encoding/json"
	"fmt"

	hamt "github.com/ipfs/go-hamt-ipld"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

func LoadState(ctx context.Context, cs *hamt.CborIpldStore, c *cid.Cid) (*State, error) {
	var n hamt.Node
	if err := cs.Get(ctx, c, &n); err != nil {
		return nil, err
	}

	return &State{root: &n, store: cs}, nil
}

type State struct {
	root  *hamt.Node
	store *hamt.CborIpldStore
}

type Actor struct {
	Code   *cid.Cid
	Memory *cid.Cid
}

func loadActor(ctx context.Context, st *hamt.Node, a Address) (*Actor, error) {
	actData, err := st.Find(ctx, string(a))
	if err != nil {
		return nil, err
	}

	var act Actor
	if err := json.Unmarshal(actData, &act); err != nil {
		return nil, fmt.Errorf("invalid account: %s", err)
	}

	return &act, nil
}

func (s *State) Copy() *State {
	return &State{
		root:  s.root.Copy(),
		store: s.store,
	}
}

func (s *State) GetActor(ctx context.Context, a Address) (*Actor, error) {
	return loadActor(ctx, s.root, a)

}

func (s *State) SetActor(ctx context.Context, a Address, act *Actor) error {
	data, err := json.Marshal(act)
	if err != nil {
		return err
	}
	if err := s.root.Set(ctx, string(a), data); err != nil {
		return err
	}

	return nil
}

func (s *State) ActorExec(ctx context.Context, addr Address, op func(*ContractState, Contract) error) error {
	act, err := s.GetActor(ctx, addr)
	if err != nil {
		return fmt.Errorf("get actor: %s", err)
	}

	contract, err := s.GetContract(ctx, act.Code)
	if err != nil {
		return fmt.Errorf("get contract: %s %s", act.Code, err)
	}

	st, err := s.LoadContractState(ctx, act.Memory)
	if err != nil {
		return fmt.Errorf("state load: %s", err)
	}

	if err := op(st, contract); err != nil {
		return err
	}

	nmemory, err := st.Flush(ctx)
	if err != nil {
		return err
	}

	act.Memory = nmemory

	if err := s.SetActor(ctx, addr, act); err != nil {
		return fmt.Errorf("set actor: %s", err)
	}

	return nil
}

func (s *State) ApplyTransactions(ctx context.Context, txs []*Transaction) error {
	for _, tx := range txs {
		err := s.ActorExec(ctx, tx.To, func(st *ContractState, contract Contract) error {
			callCtx := &CallContext{State: s, From: tx.From, Ctx: ctx, ContractState: st}
			_, err := contract.Call(callCtx, tx.Method, tx.Params)
			return err
		})
		if err != nil {
			// TODO: if the contract execution above fails, return special
			// error such that the state isnt updated, but we continue here
			return err
		}
	}

	return nil
}

func (s *State) Flush(ctx context.Context) (*cid.Cid, error) {
	if err := s.root.Flush(ctx); err != nil {
		return nil, err
	}

	return s.store.Put(ctx, s.root)
}

func (s *State) NewContractState() *ContractState {
	return &ContractState{hamt.NewNode(s.store), s.store}
}

func (s *State) LoadContractState(ctx context.Context, mem *cid.Cid) (*ContractState, error) {
	var n hamt.Node
	if err := s.store.Get(ctx, mem, &n); err != nil {
		return nil, fmt.Errorf("store get: %s", err)
	}

	return &ContractState{&n, s.store}, nil
}

// Actually, this probably should take a cid, not an address
func (s *State) GetContract(ctx context.Context, codeHash *cid.Cid) (Contract, error) {
	switch {
	case codeHash.Equals(FilecoinContractCid):
		return new(FilecoinTokenContract), nil
	case codeHash.Equals(StorageContractCodeCid):
		return new(StorageContract), nil
	case codeHash.Equals(MinerContractCodeHash):
		return new(MinerContract), nil
	default:
		return nil, fmt.Errorf("no contract code found")
	}
}

type ContractState struct {
	n    *hamt.Node
	cstr *hamt.CborIpldStore
}

func (cs *ContractState) Flush(ctx context.Context) (*cid.Cid, error) {
	if err := cs.n.Flush(ctx); err != nil {
		return nil, err
	}

	return cs.cstr.Put(ctx, cs.n)
}

func (cs *ContractState) Get(ctx context.Context, k string) ([]byte, error) {
	return cs.n.Find(ctx, k)
}

func (cs *ContractState) Set(ctx context.Context, k string, val []byte) error {
	return cs.n.Set(ctx, k, val)
}
