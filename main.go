package main

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/playground/go-filecoin/libp2p"
	ipfsaddr "github.com/ipfs/go-ipfs-addr"
	logging "github.com/ipfs/go-log"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("filecoin")

func main() {
	h, err := libp2p.Construct(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	fcn, err := NewFilecoinNode(h)
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 {
		a, err := ipfsaddr.ParseString(os.Args[1])
		if err != nil {
			panic(err)
		}
		err = h.Connect(context.Background(), pstore.PeerInfo{
			ID:    a.ID(),
			Addrs: []ma.Multiaddr{a.Transport()},
		})
		if err != nil {
			panic(err)
		}
		fmt.Println("Connected to other peer!")
	}

	for _, a := range h.Addrs() {
		fmt.Println(a.String() + "/ipfs/" + h.ID().Pretty())
	}

	_ = fcn
	select {}
}
