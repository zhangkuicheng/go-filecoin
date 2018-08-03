package chain

import "gx/ipfs/QmXJkSRxXHeAGmQJENct16anrKZHNECbmUoC7hMuCjLni6/go-hamt-ipld"

type DefaultSyncer struct {
	// blockstore is the raw on storage for blocks (connect to the network with bitswap)
	blockstore *hamt.CborIpldStore
}

func NewDefaultSyncer(blockstore *hamt.CborIpldStore) DefaultSyncer {
	return DefaultSyncer{
		blockstore: blockstore,
	}
}
