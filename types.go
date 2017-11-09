package main

import (
	"math"
	"math/big"

	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/libp2p/go-libp2p-peer"
)

type Block struct {
	Parent *cid.Cid

	// could use a hamt for these, really only need a set. a merkle patricia
	// trie (or radix trie, to be technically correct, thanks @wanderer) is
	// inefficient due to all the branch nodes
	Txs *cid.Cid

	// Height is the chain height of this block
	Height uint64

	// StateRoot should be a HAMT, its a very efficient KV store
	StateRoot *cid.Cid

	TickLimit *big.Int
	TicksUsed *big.Int
}

// Score returns a score for this block. This is used to choose the best chain.
func (b *Block) Score() uint64 {
	return b.Height
}

func (b *Block) Cid() *cid.Cid {
	// godawful cbor package
	data, err := cbor.DumpObject(b)
	if err != nil {
		panic(err)
	}
	nd, err := cbor.Decode(data, math.MaxUint64, -1)
	if err != nil {
		panic(err)
	}

	return nd.Cid()
}

// Signature over a transaction, like how ethereum does it
// TODO: think about how the signature could be an object that wraps the base
// transaction. This might make content addressing a little bit simpler
type Signature struct {
	V *big.Int
	R *big.Int
	S *big.Int
}

// Message is the equivalent of an ethereum transaction. But since ethereum
// transactions arent really transactions, theyre more just sending information
// from A to B, I (along with wanderer) want to call it a message. At the same
// time, we want to rename gas to ticks, since what the hell is gas anyways?
type Message struct {
	Nonce    uint64
	TickCost *big.Int
	Ticks    *big.Int
	To       peer.ID // actually, should be an 'account' not a peer
	Value    *big.Int
	Data     []byte

	Signature *Signature
}

// TODO: we could control creation of transaction instances to guarantee this
// never errors. Pretty annoying to do though
func (tx *Message) Cid() (*cid.Cid, error) {
	panic("NYI")
}
