package main

import (
	"encoding/json"
	"math/big"

	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	dag "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/merkledag"
)

type Block struct {
	Parent *cid.Cid

	// could use a hamt for these, really only need a set. a merkle patricia
	// trie (or radix trie, to be technically correct, thanks @wanderer) is
	// inefficient due to all the branch nodes
	// the important thing here is minizing the number of intermediate nodes
	// the simplest thing might just be to use a sorted list of cids for now.
	// A simple array can fit over 5000 cids in a single 256k block.
	//Txs *cid.Cid
	Txs []*Transaction

	// Height is the chain height of this block
	Height uint64

	// StateRoot should be a HAMT, its a very efficient KV store
	StateRoot *cid.Cid

	TickLimit *big.Int
	TicksUsed *big.Int
}

type Address string

// Score returns a score for this block. This is used to choose the best chain.
func (b *Block) Score() uint64 {
	return b.Height
}

func (b *Block) Cid() *cid.Cid {
	return b.ToNode().Cid()
}

func (b *Block) ToNode() *dag.RawNode {
	// TODO: really, anything but this. stop. please.
	data, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}

	return dag.NewRawNode(data)
}

// Signature over a transaction, like how ethereum does it
// TODO: think about how the signature could be an object that wraps the base
// transaction. This might make content addressing a little bit simpler
type Signature struct {
	V *big.Int
	R *big.Int
	S *big.Int
}

// Transaction is the equivalent of an ethereum transaction. But since ethereum
// transactions arent really transactions, theyre more just sending information
// from A to B, I (along with wanderer) want to call it a message. At the same
// time, we want to rename gas to ticks, since what the hell is gas anyways?
type Transaction struct {
	Nonce    uint64
	TickCost *big.Int
	Ticks    *big.Int
	To       Address
	Value    *big.Int
	Data     []byte

	Signature *Signature
}

// TODO: we could control creation of transaction instances to guarantee this
// never errors. Pretty annoying to do though
func (tx *Transaction) Cid() (*cid.Cid, error) {
	panic("NYI")
}
