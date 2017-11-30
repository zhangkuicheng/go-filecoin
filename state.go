package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

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

func loadAccount(ctx context.Context, st *hamt.Node, a Address) (*Account, error) {
	accData, err := st.Find(ctx, string(a))
	switch err {
	case hamt.ErrNotFound:
		// no account exists, return empty account
		return &Account{Balance: big.NewInt(0)}, nil
	default:
		return nil, err
	case nil:
		// noop
	}

	var acc Account
	if err := json.Unmarshal(accData, &acc); err != nil {
		return nil, fmt.Errorf("invalid account: %s", err)
	}

	return &acc, nil
}

func (s *State) Copy() *State {
	return &State{
		root:  s.root.Copy(),
		store: s.store,
	}
}

func (s *State) GetAccount(ctx context.Context, a Address) (*Account, error) {
	return loadAccount(ctx, s.root, a)

}

func (s *State) UpdateAccount(ctx context.Context, a Address, acc *Account) error {
	accDataOut, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	if err := s.root.Set(ctx, string(a), accDataOut); err != nil {
		return err
	}

	return nil
}

func (s *State) ApplyTransactions(ctx context.Context, txs []*Transaction) error {
	for i, tx := range txs {
		miningReward := (i == 0 && tx.Value.Cmp(MiningReward) == 0)
		if !miningReward {
			acc, err := s.GetAccount(ctx, tx.FROMTEMP)
			if err != nil {
				return err
			}

			// TODO: account for transaction fees
			if acc.Balance.Cmp(tx.Value) < 0 {
				return fmt.Errorf("not enough funds for transaction")
			}

			acc.Balance = acc.Balance.Sub(acc.Balance, tx.Value)

			if err := s.UpdateAccount(ctx, tx.FROMTEMP, acc); err != nil {
				return err
			}
		}

		toacc, err := s.GetAccount(ctx, tx.To)
		if err != nil {
			return err
		}

		toacc.Balance = toacc.Balance.Add(toacc.Balance, tx.Value)

		if err := s.UpdateAccount(ctx, tx.To, toacc); err != nil {
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
