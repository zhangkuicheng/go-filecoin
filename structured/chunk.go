package structured

import (
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"gx/ipfs/QmdiJeCpVznoeWgQjZ2N8n5ngN3GzzVABaa7Rv9vosSPUw/noms/go/chunks"
)

// Chunk is a raw chunk of immutable binary data.
// Chunks are expected to be on the order of 4KB, but they can be any size.
type Chunk struct {
	c chunks.Chunk
}

// Cid returns the Cid that globally identifies this chunk.
func (c Chunk) Cid() *cid.Cid {
	if c.Empty() {
		return nil
	}
	return mustHashToCid(c.c.Hash())
}

// Empty returns true if the chunk contains no data.
func (c Chunk) Empty() bool {
	return c.c.IsEmpty()
}

// Data returns the data the chunk contains.
func (c Chunk) Data() []byte {
	if c.Empty() {
		return nil
	}
	return c.c.Data()
}

// NewChunk creates a new chunk from some data.
func NewChunk(data []byte) Chunk {
	return Chunk{
		c: chunks.NewChunk(data),
	}
}
