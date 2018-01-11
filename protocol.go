package main

import (
	"context"
	"encoding/json"

	"gx/ipfs/QmU4vCDZTPLDqSDKguWbHCiUe46mZUtmM2g2suBZ9NE8ko/go-libp2p-net"
	ma "gx/ipfs/QmW8s4zTsUoX1Q6CeYxVKPyqSKbF7H1YDUyTostBtZ8DaG/go-multiaddr"
	"gx/ipfs/QmWNY7dV54ZDYmTA1ykVdwNCqC11mpU4zSUp6XDpLTH9eG/go-libp2p-peer"
	"gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
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
	s, err := fcn.h.NewStream(ctx, p, HelloProtocol)
	if err != nil {
		log.Error("failed to open stream to new peer for hello: ", err)
		return
	}
	defer s.Close()

	hello := &HelloMessage{
		Head:        fcn.stateMgr.headCid,
		BlockHeight: fcn.stateMgr.bestBlock.Score(),
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

	if hello.BlockHeight <= fcn.stateMgr.bestBlock.Score() {
		return
	}

	var blk Block
	if err := fcn.cs.Get(context.Background(), hello.Head, &blk); err != nil {
		log.Error("getting block from hello message: ", err)
		return
	}
	fcn.stateMgr.Inform(s.Conn().RemotePeer(), &blk)
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
