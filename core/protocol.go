package core

import (
	"context"
	"encoding/json"

	"gx/ipfs/QmQm7WmgYCa4RSz76tKEYpRjApjnRw8ZTUVQC15b8JM4a2/go-libp2p-net"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	"gx/ipfs/Qma7H6RW8wRrfZpNSXwxYGcd1E149s42FpWNpDNieSVrnU/go-libp2p-peer"
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	types "github.com/filecoin-project/go-filecoin/types"
)

var TxsTopic = "/fil/tx"
var BlocksTopic = "/fil/blks"

const HelloProtocol = "/filecoin/hello/0.0.1"

type HelloMessage struct {
	// Could just put the full block header here
	Head        *cid.Cid
	BlockHeight uint64

	// Maybe add some other info to gossip
}

func (fcn *FilecoinNode) HelloPeer(p peer.ID) {
	ctx := context.Background() // TODO: add appropriate timeout
	s, err := fcn.Host.NewStream(ctx, p, HelloProtocol)
	if err != nil {
		log.Error("failed to open stream to new peer for hello: ", err)
		return
	}
	defer s.Close()

	hello := &HelloMessage{
		Head:        fcn.StateMgr.HeadCid,
		BlockHeight: fcn.StateMgr.BestBlock.Score(),
	}

	if err := json.NewEncoder(s).Encode(hello); err != nil {
		log.Error("marshaling hello message to new peer: ", err)
		return
	}
}

func (fcn *FilecoinNode) handleHelloStream(s net.Stream) {
	var hello HelloMessage
	if err := json.NewDecoder(s).Decode(&hello); err != nil {
		log.Error("decoding hello message: ", err)
		return
	}

	if hello.Head == nil {
		return
	}

	if hello.BlockHeight <= fcn.StateMgr.BestBlock.Score() {
		return
	}

	var blk types.Block
	if err := fcn.cs.Get(context.Background(), hello.Head, &blk); err != nil {
		log.Error("getting block from hello message: ", err)
		return
	}
	fcn.StateMgr.Inform(s.Conn().RemotePeer(), &blk)
}

type fcnNotifiee FilecoinNode

var _ net.Notifiee = (*fcnNotifiee)(nil)

func (n *fcnNotifiee) ClosedStream(_ net.Network, s net.Stream) {}
func (n *fcnNotifiee) OpenedStream(_ net.Network, s net.Stream) {}
func (n *fcnNotifiee) Connected(_ net.Network, c net.Conn) {
	go (*FilecoinNode)(n).HelloPeer(c.RemotePeer())
}

func (n *fcnNotifiee) Disconnected(_ net.Network, c net.Conn)    {}
func (n *fcnNotifiee) Listen(_ net.Network, a ma.Multiaddr)      {}
func (n *fcnNotifiee) ListenClose(_ net.Network, a ma.Multiaddr) {}
