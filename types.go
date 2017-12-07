package main

import (
	"encoding/hex"
	"encoding/json"
	"math/big"

	cbor "github.com/ipfs/go-ipld-cbor"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

func init() {
	cbor.RegisterCborType(Block{})
	cbor.RegisterCborType(Transaction{})
	cbor.RegisterCborType(Signature{})
}

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

func (a Address) String() string {
	return "0x" + hex.EncodeToString([]byte(a))
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Address) UnmarshalJSON(b []byte) error {
	b = b[3 : len(b)-1]
	outbuf := make([]byte, len(b)/2)
	_, err := hex.Decode(outbuf, b)
	if err != nil {
		return err
	}

	*a = Address(outbuf)
	return nil
}

func ParseAddress(s string) (Address, error) {
	if s[:2] == "0x" {
		s = s[2:]
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}

	return Address(b), nil
}

// Score returns a score for this block. This is used to choose the best chain.
func (b *Block) Score() uint64 {
	return b.Height
}

func (b *Block) Cid() *cid.Cid {
	return b.ToNode().Cid()
}

func (b *Block) ToNode() node.Node {
	obj, err := cbor.WrapObject(b, mh.SHA2_256, -1)
	if err != nil {
		panic(err)
	}

	return obj
}

func DecodeBlock(b []byte) (*Block, error) {
	var out Block
	if err := cbor.DecodeInto(b, &out); err != nil {
		return nil, err
	}

	return &out, nil
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
// TODO: ensure that transactions are not malleable *at all*, this is very important
type Transaction struct {
	To   Address // address of contract to invoke
	From Address

	Nonce uint64

	TickCost *big.Int
	Ticks    *big.Int
	Method   string
	Params   []interface{}

	Signature *Signature
}

// TODO: we could control creation of transaction instances to guarantee this
// never errors. Pretty annoying to do though
func (tx *Transaction) Cid() (*cid.Cid, error) {
	obj, err := cbor.WrapObject(tx, mh.SHA2_256, -1)
	if err != nil {
		return nil, err
	}

	return obj.Cid(), nil
}
