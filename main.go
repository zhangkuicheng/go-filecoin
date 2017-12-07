package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	ipfsaddr "gx/ipfs/QmUnAfDeH1Nths56yuMvkw4V3sNF4d1xBbWy5hfZG7LF6G/go-ipfs-addr"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"

	cli "github.com/urfave/cli"

	"github.com/filecoin-project/playground/go-filecoin/libp2p"

	"gx/ipfs/QmVNv1WV6XxzQV4MBuiLX5729wMazaf8TNzm2Sq6ejyHh7/go-libp2p-floodsub"

	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"

	bstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	dag "github.com/ipfs/go-ipfs/merkledag"
	none "github.com/ipfs/go-ipfs/routing/none"
)

var log = logging.Logger("filecoin")

var daemonCmd = cli.Command{
	Name: "daemon",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name: "no-api",
		},
		cli.StringFlag{
			Name:  "api",
			Value: ":3453",
		},
	},
	Action: func(c *cli.Context) error {
		h, err := libp2p.Construct(context.Background(), nil)
		if err != nil {
			panic(err)
		}

		fsub := floodsub.NewFloodSub(context.Background(), h)

		// Also should probably pass in the dagservice instance
		bs := bstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore()))
		nilr, _ := none.ConstructNilRouting(nil, nil, nil)
		bsnet := bsnet.NewFromIpfsHost(h, nilr)
		bswap := bitswap.New(context.Background(), h.ID(), bsnet, bs, true)
		bserv := bserv.New(bs, bswap)
		dag := dag.NewDAGService(bserv)

		fcn, err := NewFilecoinNode(h, fsub, dag, bserv, bswap.(*bitswap.Bitswap))
		if err != nil {
			panic(err)
		}

		if c.Args().Present() {
			a, err := ipfsaddr.ParseString(c.Args().First())
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
		fmt.Println("C ARGS: ", c.Args())

		for _, a := range h.Addrs() {
			fmt.Println(a.String() + "/ipfs/" + h.ID().Pretty())
		}

		if c.Bool("no-api") {
			select {}
		}

		ba := NewBadApi(fcn)
		http.HandleFunc("/api", ba.ApiHandlerPleaseReplace)

		panic(http.ListenAndServe(c.String("api"), nil))
	},
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		daemonCmd,
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "api",
			Value: ":3453",
		},
	}
	app.Action = func(c *cli.Context) error {
		rpc := &RPC{
			Method: c.Args().First(),
			Args:   c.Args().Tail(),
		}

		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(rpc); err != nil {
			panic(err)
		}
		resp, err := http.Post(fmt.Sprintf("http://localhost%s/api", c.String("api")), "application/json", buf)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			msg, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			return fmt.Errorf("%s: %s", resp.Status, string(msg))
		}

		var i interface{}
		if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
			return err
		}
		fmt.Println(i)
		return nil
	}

	app.RunAndExitOnError()
}
