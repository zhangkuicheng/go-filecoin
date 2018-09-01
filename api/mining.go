package api

import (
	"context"

	"github.com/filecoin-project/go-filecoin/chain"
)

// Mining is the interface that defines methods to manage mining operations.
type Mining interface {
	Once(ctx context.Context) (*chain.Block, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
