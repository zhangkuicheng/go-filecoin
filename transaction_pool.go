package main

import (
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
)

type TransactionPool struct {
	// TODO: an in memory set is probably not the right thing to use here, but whatever
	txset *cid.Set
}

func (txp *TransactionPool) Add(tx *Transaction) error {
	c, err := tx.Cid()
	if err != nil {
		return err
	}

	txp.txset.Add(c)
	return nil
}
