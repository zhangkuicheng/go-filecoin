package main

import (
	"context"
	"fmt"
	"math/big"
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

	// address is the address that the miner will mine rewards to
	address Address

	// transaction pool to pull transactions for the next block from
	txPool *TransactionPool

	// cheating.
	fcn *FilecoinNode
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

var MiningReward = big.NewInt(1000000)

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
			nb, err := m.getNextBlock(ctx)
			if err != nil {
				log.Error("failed to build block on top of: ", m.currentBlock.Cid())
				log.Error(err)
				break
			}

			fmt.Printf("==> mined a new block [score %d, %s] in %s [av: %s]\n", nb.Score(), nb.Cid(), time.Since(start), time.Since(begin)/n)

			if err := m.blockCallback(nb); err != nil {
				log.Error("mined new block, but failed to push it out: ", err)
				break
			}
			m.currentBlock = nb
		}
		blockFound.Reset(predictFuture())
	}
}

func (m *Miner) getNextBlock(ctx context.Context) (*Block, error) {
	reward := &Transaction{
		From:   FilecoinContractAddr,
		To:     FilecoinContractAddr,
		Method: "transfer",
		Params: []interface{}{m.address, MiningReward},
	}

	fmt.Println("mining to: ", m.address)

	txs := m.txPool.GetBestTxs()
	txs = append([]*Transaction{reward}, txs...)
	nb := &Block{
		Height: m.currentBlock.Height + 1,
		Parent: m.currentBlock.Cid(),
		Txs:    txs,
	}

	s, err := LoadState(ctx, m.fcn.cs, m.currentBlock.StateRoot)
	if err != nil {
		return nil, err
	}

	if err := s.ApplyTransactions(ctx, nb.Txs); err != nil {
		return nil, fmt.Errorf("applying state from newly mined block: %s", err)
	}
	stateCid, err := s.Flush(ctx)
	if err != nil {
		return nil, fmt.Errorf("flushing state changes: %s", err)
	}

	nb.StateRoot = stateCid
	return nb, nil
}
