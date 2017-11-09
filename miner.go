package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

var rr = rand.New(rand.NewSource(time.Now().UnixNano()))

type Miner struct {
	// newBlocks is a channel that listens for new blocks from other peers in
	// the network
	newBlocks chan *Block

	// blockCallback is a function for the miner to call when it has mined a
	// new block
	blockCallback func(*Block) error

	// currentBlock is the block that the miner will be mining on top of
	currentBlock *Block
}

// predictFuture reads an oracle that tells us how long we must wait to mine
// the next block
func predictFuture() time.Duration {
	v := time.Hour
	for v > time.Second*10 || v < 0 {
		v = time.Duration((rr.NormFloat64()*3000)+4000) * time.Millisecond
	}
	return v
}

func (m *Miner) Run(ctx context.Context) {
	blockFound := time.NewTimer(predictFuture())

	begin := time.Now()
	for n := time.Duration(1); true; n++ {
		start := time.Now()
		select {
		case <-ctx.Done():
			log.Error("mining canceled: ", ctx.Err())
			return
		case b := <-m.newBlocks:
			m.currentBlock = b
			fmt.Printf("got a new block in %s [av: %s]\n", time.Since(start), time.Since(begin)/n)
		case <-blockFound.C:
			nb := &Block{
				Height: m.currentBlock.Height + 1,
				Parent: m.currentBlock.Cid(),
			}

			fmt.Printf("mined a new block [score %d] in %s [av: %s]\n", nb.Score(), time.Since(start), time.Since(begin)/n)

			m.currentBlock = nb
			if err := m.blockCallback(nb); err != nil {
				log.Error("mined new block, but failed to push it out: ", err)
			}
		}
		blockFound.Reset(predictFuture())
	}
}
