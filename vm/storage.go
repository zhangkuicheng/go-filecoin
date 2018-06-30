package vm

import (
	"github.com/filecoin-project/go-filecoin/exec"
	"github.com/filecoin-project/go-filecoin/types"

	cbor "gx/ipfs/QmRiRJhn427YVuufBEHofLreKWNw7P7BWNq86Sb9kzqdbd/go-ipld-cbor"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// Stage is a place to hold chunks that are created while processing a block.
type Stage map[string]ipld.Node

// storage implements exec.Storage
type storage struct {
	s Stage
}

func (s storage) Put(chunk []byte) (*cid.Cid, exec.ErrorCode) {
	// TODO: Re-parsing this is so dumb - the interface should be ipld.Node.
	n, err := cbor.Decode(chunk, types.DefaultHashFunction, -1)
	if err != nil {
		return nil, exec.ErrDecode
	}

	cid := n.Cid()
	s.s[cid.String()] = n
	return cid, exec.Ok
}

func (s storage) Get(cid *cid.Cid) ([]byte, bool) {
	n, ok := s.s[cid.String()]
	if !ok {
		return nil, ok
	}
	return n.RawData(), ok
}
