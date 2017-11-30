package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	ipfsaddr "gx/ipfs/QmUnAfDeH1Nths56yuMvkw4V3sNF4d1xBbWy5hfZG7LF6G/go-ipfs-addr"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"

	cli "github.com/urfave/cli"

	"github.com/filecoin-project/playground/go-filecoin/libp2p"

	"gx/ipfs/QmVNv1WV6XxzQV4MBuiLX5729wMazaf8TNzm2Sq6ejyHh7/go-libp2p-floodsub"

	bstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	dag "github.com/ipfs/go-ipfs/merkledag"
	none "github.com/ipfs/go-ipfs/routing/none"
	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
)

type RPC struct {
	Method string
	Args   []string
}

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
		bs := bstore.NewBlockstore(ds.NewMapDatastore())
		nilr, _ := none.ConstructNilRouting(nil, nil, nil)
		bsnet := bsnet.NewFromIpfsHost(h, nilr)
		bswap := bitswap.New(context.Background(), h.ID(), bsnet, bs, true)
		bserv := bserv.New(bs, bswap)
		dag := dag.NewDAGService(bserv)

		fcn, err := NewFilecoinNode(h, fsub, dag, bserv)
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

		http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
			// TODO: don't use a json rpc. it sucks. but its easy.
			var rpc RPC
			if err := json.NewDecoder(r.Body).Decode(&rpc); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			var out interface{}
			switch rpc.Method {
			case "listAddrs":
				out = fcn.Addresses
			case "newAddr":
				naddr := fcn.createNewAddress()
				fcn.Addresses = append(fcn.Addresses, naddr)
				out = naddr
			case "sendTx":
				if len(rpc.Args) != 2 {
					out = fmt.Errorf("must pass two arguments")
					break
				}
				amount, ok := big.NewInt(0).SetString(rpc.Args[0], 10)
				if !ok {
					out = fmt.Errorf("failed to parse amount")
					break
				}
				toaddr, err := ParseAddress(rpc.Args[1])
				if err != nil {
					out = err
					break
				}

				tx := &Transaction{
					FROMTEMP: fcn.Addresses[0],
					To:       toaddr,
					Value:    amount,
				}

				fcn.SendNewTransaction(tx)
			case "getBalance":
				if len(rpc.Args) != 1 {
					out = fmt.Errorf("must pass address as argument")
					break
				}

				addr, err := ParseAddress(rpc.Args[0])
				if err != nil {
					out = err
					break
				}

				acc, err := fcn.stateRoot.GetAccount(context.Background(), addr)
				if err != nil {
					out = err
					break
				}

				out = acc.Balance.String()
			}

			json.NewEncoder(w).Encode(out)
		})

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

		var i interface{}
		if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
			return err
		}
		fmt.Println(i)
		return nil
	}

	app.RunAndExitOnError()
}
