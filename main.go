package main

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/playground/go-filecoin/libp2p"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	ipfsaddr "gx/ipfs/QmeS8cCKawUwejVrsBtmC1toTXmwVWZGiRJqzgTURVWeF9/go-ipfs-addr"
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
