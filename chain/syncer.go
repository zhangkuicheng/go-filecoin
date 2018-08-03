package chain

import (
	"context"

	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

type Syncer interface {
	HandleNewBlocksFromNetwork(ctx context.Context, cids []*cid.Cid, height uint64)
}
