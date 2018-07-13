package structured

import (
	"fmt"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/filecoin-project/go-filecoin/util/chk"

	"github.com/filecoin-project/go-filecoin/types"

	"github.com/attic-labs/noms/go/hash"
)

func hashToCid(h hash.Hash) (*cid.Cid, error) {
	if h.IsEmpty() {
		return nil, nil
	}
	mh, err := mh.Encode(h[:], mh.SHA2_512)
	if err != nil {
		return nil, errors.Wrap(err, "Could not encode multihash from Noms hash")
	}

	return cid.NewCidV1(types.NomsMulticodec, mh), nil
}

func mustHashToCid(h hash.Hash) *cid.Cid {
	cid, err := hashToCid(h)
	chk.True(err == nil)
	return cid
}

func cidToHash(c *cid.Cid) (h hash.Hash, err error) {
	if c == nil {
		return h, nil
	}
	if c.Type() != types.NomsMulticodec {
		return h, fmt.Errorf("Invalid multicodec type %x - must be Noms (%x)", c.Type(), types.NomsMulticodec)
	}
	dmh, err := mh.Decode(c.Hash())
	if err != nil {
		return h, errors.Wrap(err, "Could not decode multihash from Cid")
	}

	if dmh.Code != mh.SHA2_512 {
		return h, fmt.Errorf("Invalid hash code %s (%x), must be SHA2_512 (%x)", mh.Codes[dmh.Code], dmh.Code, mh.SHA2_512)
	}

	if dmh.Length != hash.ByteLen {
		return h, fmt.Errorf("Invalid hash length %d, must be %d", dmh.Length, hash.ByteLen)
	}

	return hash.New(dmh.Digest), nil
}

func mustCidToHash(c *cid.Cid) hash.Hash {
	h, err := cidToHash(c)
	chk.True(err == nil)
	return h
}
