package main

import (
	"context"
	"encoding/json"
	"fmt"

	hamt "github.com/ipfs/go-hamt-ipld"
	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
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
	Code   Address // actually should be a hash
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

func (s *State) ApplyTransactions(ctx context.Context, txs []*Transaction) error {
	for _, tx := range txs {
		act, err := s.GetActor(ctx, tx.To)
		if err != nil {
			return fmt.Errorf("get actor: %s", err)
		}

		contract, err := s.LoadContract(ctx, act.Code)
		if err != nil {
			return fmt.Errorf("load contract: %s %s", act.Code, err)
		}

		// TODO: 'state transaction'
		var ctrState hamt.Node
		if err := s.store.Get(ctx, act.Memory, &ctrState); err != nil {
			return fmt.Errorf("load contract state: %s", err)
		}

		if err := contract.LoadState(&ctrState); err != nil {
			return err
		}

		callCtx := &CallContext{Ctx: ctx, From: tx.FROMTEMP}
		_, err = contract.Call(callCtx, tx.Method, tx.Params)
		if err != nil {
			fmt.Println("call error: ", err)
			return err
		}

		if err := ctrState.Flush(ctx); err != nil {
			return err
		}

		nmemory, err := s.store.Put(ctx, ctrState)
		if err != nil {
			return err
		}

		act.Memory = nmemory

		if err := s.SetActor(ctx, tx.To, act); err != nil {
			return fmt.Errorf("set actor: %s", err)
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

// Actually, this probably should take a cid, not an address
func (s *State) LoadContract(ctx context.Context, codeHash Address) (Contract, error) {
	act, err := s.GetActor(ctx, codeHash)
	if err != nil {
		return nil, fmt.Errorf("get actor: %s", err)
	}

	var n hamt.Node
	if err := s.store.Get(ctx, act.Memory, &n); err != nil {
		return nil, fmt.Errorf("store get: %s", err)
	}

	switch act.Code {
	case FilecoinContractAddr:
		return &FilecoinTokenContract{s: &n}, nil
	default:
		return nil, fmt.Errorf("no contract code found")
	}
}
