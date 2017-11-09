package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

var rr = rand.New(rand.NewSource(time.Now().UnixNano()))

type Miner struct {
	newBlocks     chan *Block
	blockCallback func(*Block) error

	currentBlock *Block
}

func predictFuture() time.Duration {
	v := time.Hour
	for v > time.Second*10 {
		v = time.Duration((rr.NormFloat64()*4000)+8000) * time.Millisecond
	}

	fmt.Println("oracle says to wait ", v)
	return v
}

func (m *Miner) Run(ctx context.Context) {

	blockFound := time.NewTimer(predictFuture())

	for {
		select {
		case <-ctx.Done():
			log.Error("mining canceled: ", ctx.Err())
			return
		case b := <-m.newBlocks:
			log.Error("new block, restarting mining process")
			m.currentBlock = b
		case <-blockFound.C:
			fmt.Println("mined a new block!")
			nb := &Block{
				Height: m.currentBlock.Height + 1,
			}

			if err := m.blockCallback(nb); err != nil {
				log.Error("mined new block, but failed to push it out: ", err)
			}
		}
		blockFound.Reset(predictFuture())
	}
}
