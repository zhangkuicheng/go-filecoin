package core

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"gx/ipfs/QmP1T1SGU6276R2MHKP2owbck37Fnzd6ZkpyNJvnG2LoTG/go-libp2p-floodsub"
	"gx/ipfs/QmP46LGWhzVZTMmt5akNNLfoV8qL4h5wTwmzQxLyDafggd/go-libp2p-host"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"

	contract "github.com/filecoin-project/playground/go-filecoin/contract"
	state "github.com/filecoin-project/playground/go-filecoin/state"
	types "github.com/filecoin-project/playground/go-filecoin/types"

	hamt "github.com/ipfs/go-hamt-ipld"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	dag "github.com/ipfs/go-ipfs/merkledag"
)

var log = logging.Logger("core")

var ProtocolID = protocol.ID("/fil/0.0.0")

type FilecoinNode struct {
	h host.Host

	Addresses []types.Address

	bsub, txsub *floodsub.Subscription
	pubsub      *floodsub.PubSub

	DAG     dag.DAGService
	Bitswap *bitswap.Bitswap
	cs      *hamt.CborIpldStore

	StateMgr *state.StateManager
}

func NewFilecoinNode(h host.Host, fs *floodsub.PubSub, dag dag.DAGService, bs bserv.BlockService, bswap *bitswap.Bitswap) (*FilecoinNode, error) {
	fcn := &FilecoinNode{
		h:       h,
		DAG:     dag,
		Bitswap: bswap,
		cs:      &hamt.CborIpldStore{bs},
	}

	s := state.NewStateManager(fcn.cs, fcn.DAG)

	fcn.StateMgr = s

	baseAddr := CreateNewAddress()
	fcn.Addresses = []types.Address{baseAddr}
	fmt.Println("my mining address is ", baseAddr)

	genesis, err := CreateGenesisBlock(fcn.cs)
	if err != nil {
		return nil, err
	}
	s.SetBestBlock(genesis)

	c, err := fcn.DAG.Add(genesis.ToNode())
	if err != nil {
		return nil, err
	}
	fmt.Println("genesis block cid is: ", c)
	s.KnownGoodBlocks.Add(c)

	st, err := contract.LoadState(context.Background(), fcn.cs, genesis.StateRoot)
	if err != nil {
		return nil, err
	}
	s.StateRoot = st

	// TODO: better miner construction and delay start until synced
	s.Miner = state.NewMiner(fcn.SendNewBlock, s.TxPool, genesis, baseAddr, fcn.cs)
	s.Miner.StateMgr = s

	// Run miner
	go s.Miner.Run(context.Background())

	txsub, err := fs.Subscribe(TxsTopic)
	if err != nil {
		return nil, err
	}

	blksub, err := fs.Subscribe(BlocksTopic)
	if err != nil {
		return nil, err
	}

	go fcn.processNewBlocks(blksub)
	go fcn.processNewTransactions(txsub)

	h.SetStreamHandler(HelloProtocol, fcn.handleHelloStream)
	h.Network().Notify((*fcnNotifiee)(fcn))

	fcn.txsub = txsub
	fcn.bsub = blksub
	fcn.pubsub = fs

	return fcn, nil
}

func (fcn *FilecoinNode) processNewTransactions(txsub *floodsub.Subscription) {
	// TODO: this function should really just be a validator function for the pubsub subscription
	for {
		msg, err := txsub.Next(context.Background())
		if err != nil {
			panic(err)
		}

		var txmsg types.Transaction
		if err := json.Unmarshal(msg.GetData(), &txmsg); err != nil {
			panic(err)
		}

		if err := fcn.StateMgr.TxPool.Add(&txmsg); err != nil {
			panic(err)
		}
	}
}

func CreateNewAddress() types.Address {
	buf := make([]byte, 20)
	rand.Read(buf)
	return types.Address(buf)
}

func (fcn *FilecoinNode) processNewBlocks(blksub *floodsub.Subscription) {
	// TODO: this function should really just be a validator function for the pubsub subscription
	for {
		msg, err := blksub.Next(context.Background())
		if err != nil {
			panic(err)
		}
		if msg.GetFrom() == fcn.h.ID() {
			continue
		}

		blk, err := types.DecodeBlock(msg.GetData())
		if err != nil {
			panic(err)
		}

		fcn.StateMgr.Inform(msg.GetFrom(), blk)
	}
}

func (fcn *FilecoinNode) SendNewBlock(b *types.Block) error {
	nd := b.ToNode()
	_, err := fcn.DAG.Add(nd)
	if err != nil {
		return err
	}

	if err := fcn.StateMgr.ProcessNewBlock(context.Background(), b); err != nil {
		return err
	}

	return fcn.pubsub.Publish(BlocksTopic, nd.RawData())
}

func (fcn *FilecoinNode) SendNewTransaction(tx *types.Transaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return fcn.pubsub.Publish(TxsTopic, data)
}
