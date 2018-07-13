package structured

import (
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/attic-labs/noms/go/chunks"
)

// Chunk is a raw chunk of binary data.
// Chunks are typically on the order of 4KB, but that is not guaranteed.
type Chunk struct {
	c chunks.Chunk
}

func (c Chunk) Cid() *cid.Cid {
	if c.Empty() {
		return nil
	}
	return mustHashToCid(c.c.Hash())
}

func (c Chunk) Empty() bool {
	return c.c.IsEmpty()
}

func (c Chunk) Data() []byte {
	if c.Empty() {
		return nil
	}
	return c.c.Data()
}

func NewChunk(data []byte) Chunk {
	return Chunk{
		c: chunks.NewChunk(data),
	}
}
