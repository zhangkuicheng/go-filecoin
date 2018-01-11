package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	ma "gx/ipfs/QmW8s4zTsUoX1Q6CeYxVKPyqSKbF7H1YDUyTostBtZ8DaG/go-multiaddr"
	pstore "gx/ipfs/QmYijbtjCxFEjSXaudaQAUz3LN5VKLssm8WCUsRoqzXmQR/go-libp2p-peerstore"
	ipfsaddr "gx/ipfs/QmdMeXVB1V1SAZcFzoCuM3zR9K8PeuzCYg4zXNHcHh6dHU/go-ipfs-addr"
	"gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"

	libp2p "github.com/filecoin-project/playground/go-filecoin/libp2p"

	"gx/ipfs/QmP1T1SGU6276R2MHKP2owbck37Fnzd6ZkpyNJvnG2LoTG/go-libp2p-floodsub"

	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"

	bstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	bserv "github.com/ipfs/go-ipfs/blockservice"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	dag "github.com/ipfs/go-ipfs/merkledag"
	path "github.com/ipfs/go-ipfs/path"
	none "github.com/ipfs/go-ipfs/routing/none"

	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdcli "github.com/ipfs/go-ipfs-cmds/cli"
	cmdhttp "github.com/ipfs/go-ipfs-cmds/http"
)

var log = logging.Logger("filecoin")

func fail(v ...interface{}) {
	fmt.Println(v)
	os.Exit(1)
}

func main() {
	daemonRunning, err := daemonIsRunning()
	if err != nil {
		fail(err)
	}

	req, err := cmdcli.Parse(context.Background(), os.Args[1:], os.Stdin, RootCmd)
	if err != nil {
		panic(err)
	}

	if daemonRunning {
		if req.Command == DaemonCmd { // this is a hack, go-ipfs does this slightly better
			fmt.Println("daemon already running...")
			return
		}
		client := cmdhttp.NewClient(":3453", cmdhttp.ClientWithAPIPrefix("/api"))

		// send request to server
		res, err := client.Send(req)
		if err != nil {
			panic(err)
		}

		encType := cmds.GetEncoding(req)
		enc := req.Command.Encoders[encType]
		if enc == nil {
			enc = cmds.Encoders[encType]
		}

		// create an emitter
		re, retCh := cmdcli.NewResponseEmitter(os.Stdout, os.Stderr, enc, req)

		if pr, ok := req.Command.PostRun[cmds.CLI]; ok {
			re = pr(req, re)
		}

		wait := make(chan struct{})
		// copy received result into cli emitter
		go func() {
			err = cmds.Copy(re, res)
			if err != nil {
				re.SetError(err, cmdkit.ErrNormal|cmdkit.ErrFatal)
			}
			close(wait)
		}()

		// wait until command has returned and exit
		ret := <-retCh
		<-wait
		os.Exit(ret)
	} else {
		req.Options[cmds.EncLong] = cmds.Text

		// create an emitter
		re, retCh := cmdcli.NewResponseEmitter(os.Stdout, os.Stderr, req.Command.Encoders[cmds.Text], req)

		if pr, ok := req.Command.PostRun[cmds.CLI]; ok {
			re = pr(req, re)
		}

		wait := make(chan struct{})
		// call command in background
		go func() {
			defer close(wait)

			err = RootCmd.Call(req, re, nil)
			if err != nil {
				panic(err)
			}
		}()

		// wait until command has returned and exit
		ret := <-retCh
		<-wait

		os.Exit(ret)
	}
}

var RootCmd = &cmds.Command{
	Options: []cmdkit.Option{
		cmdkit.StringOption("api", "set the api port to use").WithDefault(":3453"),
		cmds.OptionEncodingType,
	},
	Subcommands: make(map[string]*cmds.Command),
}

var rootSubcommands = map[string]*cmds.Command{
	"daemon":  DaemonCmd,
	"addrs":   AddrsCmd,
	"bitswap": BitswapCmd,
	"dag":     DagCmd,
	"wallet":  WalletCmd,
	"order":   OrderCmd,
	"miner":   MinerCmd,
}

func init() {
	for k, v := range rootSubcommands {
		RootCmd.Subcommands[k] = v
	}
}

var DaemonCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "run the filecoin daemon",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("bootstrap", false, true, "nodes to bootstrap to"),
	},
	Run: daemonRun,
}

func daemonRun(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
	// set up networking
	h, err := libp2p.Construct(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	fsub := floodsub.NewFloodSub(context.Background(), h)

	// set up storage (a bit more complicated than it realistically needs to be right now)
	bs := bstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore()))
	nilr, _ := none.ConstructNilRouting(nil, nil, nil)
	bsnet := bsnet.NewFromIpfsHost(h, nilr)
	bswap := bitswap.New(context.Background(), h.ID(), bsnet, bs, true)
	bserv := bserv.New(bs, bswap)
	dag := dag.NewDAGService(bserv)

	// TODO: work on what parameters we pass to the filecoin node
	fcn, err := NewFilecoinNode(h, fsub, dag, bserv, bswap.(*bitswap.Bitswap))
	if err != nil {
		panic(err)
	}

	if len(req.Arguments) > 0 {
		a, err := ipfsaddr.ParseString(req.Arguments[0])
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

	if err := writeDaemonLock(); err != nil {
		panic(err)
	}

	servenv := &CommandEnv{
		ctx:  context.Background(),
		Node: fcn,
	}

	cfg := cmdhttp.NewServerConfig()
	cfg.APIPath = "/api"

	handler := cmdhttp.NewHandler(servenv, RootCmd, cfg)

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)

	api := req.Options["api"].(string)

	go func() {
		panic(http.ListenAndServe(api, handler))
	}()

	<-ch
	removeDaemonLock()
}

type CommandEnv struct {
	ctx  context.Context
	Node *FilecoinNode
}

func (ce *CommandEnv) Context() context.Context {
	return ce.ctx
}

func GetNode(env cmds.Environment) *FilecoinNode {
	ce := env.(*CommandEnv)
	return ce.Node
}

var AddrsCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"new":  AddrsNewCmd,
		"list": AddrsListCmd,
	},
}

var AddrsNewCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		naddr := createNewAddress()
		fcn.Addresses = append(fcn.Addresses, naddr)
		re.Emit(naddr)
	},
	Type: Address(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			a, ok := v.(*Address)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			_, err := fmt.Fprintln(w, a.String())
			return err
		}),
	},
}

var AddrsListCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		re.Emit(fcn.Addresses)
	},
	Type: []Address{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			addrs, ok := v.(*[]Address)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			for _, a := range *addrs {
				_, err := fmt.Fprintln(w, a.String())
				if err != nil {
					return err
				}
			}
			return nil
		}),
	},
}

var BitswapCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"wantlist": BitswapWantlistCmd,
	},
}

var BitswapWantlistCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		re.Emit(fcn.bswap.GetWantlist())
	},
	Type: []*cid.Cid{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			wants, ok := v.(*[]cid.Cid)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			for _, want := range *wants {
				_, err := fmt.Fprintln(w, want.String())
				if err != nil {
					return err
				}
			}
			return nil
		}),
	},
}

var DagCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"get": DagGetCmd,
	},
}

var DagGetCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("object", true, false, "ref of node to get"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)

		res := path.NewBasicResolver(fcn.dag)

		p, err := path.ParsePath(req.Arguments[0])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		nd, err := res.ResolvePath(req.Context, p)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		re.Emit(nd)
	},
}

var WalletCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"send":    WalletSendCmd,
		"balance": WalletGetBalanceCmd,
	},
}

var WalletGetBalanceCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("account", false, false, "account to get balance of"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)

		addr := fcn.Addresses[0]
		if len(req.Arguments) > 0 {
			a, err := ParseAddress(req.Arguments[0])
			if err != nil {
				re.SetError(err, cmdkit.ErrNormal)
				return
			}
			addr = a
		}

		stroot := fcn.stateMgr.stateRoot
		act, err := stroot.GetActor(req.Context, FilecoinContractAddr)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		ct, err := stroot.GetContract(req.Context, act.Code)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		cst, err := stroot.LoadContractState(req.Context, act.Memory)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		cctx := &CallContext{Ctx: req.Context, ContractState: cst}
		val, err := ct.Call(cctx, "getBalance", []interface{}{addr})
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		re.Emit(val)
	},
	Type: big.Int{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			val, ok := v.(*big.Int)
			if !ok {
				return fmt.Errorf("got unexpected type: %T", v)
			}
			fmt.Fprintln(w, val.String())
			return nil
		}),
	},
}

// TODO: this command should exist in some form, but its really specialized.
// The issue is that its really 'call transfer on the filecoin token contract
// and send tokens from our default account to a given actor'.
var WalletSendCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("value", true, false, "amount to send"),
		cmdkit.StringArg("to", true, false, "actor to send transaction to"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)

		amount, ok := big.NewInt(0).SetString(req.Arguments[0], 10)
		if !ok {
			re.SetError("failed to parse amount", cmdkit.ErrNormal)
			return
		}
		toaddr, err := ParseAddress(req.Arguments[1])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		tx := &Transaction{
			From:   fcn.Addresses[0],
			To:     FilecoinContractAddr,
			Method: "transfer",
			Params: []interface{}{toaddr, amount},
		}

		fcn.SendNewTransaction(tx)
	},
}

var MinerCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"new": MinerNewCmd,
	},
}

var MinerNewCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("pledge-size", true, false, "size of pledge to create miner with"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)

		pledge, ok := big.NewInt(0).SetString(req.Arguments[0], 10)
		if !ok {
			re.SetError("failed to parse pledge as number", cmdkit.ErrNormal)
			return
		}

		tx := &Transaction{
			From:   fcn.Addresses[0],
			To:     StorageContractAddress,
			Method: "createMiner",
			Params: []interface{}{pledge},
		}

		if err := fcn.SendNewTransaction(tx); err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		// TODO: wait for tx to be mined, read receipts for address
	},
}

var OrderCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"bid": OrderBidCmd,
		"ask": OrderAskCmd,
	},
}

var OrderBidCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"add":  OrderBidAddCmd,
		"list": OrderBidListCmd,
	},
}

var OrderBidAddCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("price", true, false, "price for bid"),
		cmdkit.StringArg("size", true, false, "size of bid"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		price, err := strconv.Atoi(req.Arguments[0])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		size, err := strconv.Atoi(req.Arguments[1])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		b := &Bid{
			Size:  uint64(size),
			Price: big.NewInt(int64(price)),
		}

		tx := &Transaction{
			From:   fcn.Addresses[0],
			To:     StorageContractAddress,
			Method: "addBid",
			Params: []interface{}{b},
		}

		if err := fcn.SendNewTransaction(tx); err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
	},
}

var OrderBidListCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		bids, err := listBids(req.Context, fcn)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		re.Emit(bids)
	},
	Type: []*Bid{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			bids, ok := v.(*[]*Bid)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			for _, b := range *bids {
				fmt.Fprintln(w, b.Owner, b.Price, b.Size, b.Collateral)
			}
			return nil
		}),
	},
}
var OrderAskCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"add":  OrderAskAddCmd,
		"list": OrderAskListCmd,
	},
}

var OrderAskAddCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("miner", true, false, "miner to create ask on"),
		cmdkit.StringArg("price", true, false, "price per byte being asked"),
		cmdkit.StringArg("size", true, false, "total size being offered"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		miner, err := ParseAddress(req.Arguments[0])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		price, err := strconv.Atoi(req.Arguments[1])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		size, err := strconv.Atoi(req.Arguments[2])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		a := &Ask{
			Size:  uint64(size),
			Price: big.NewInt(int64(price)),
		}

		tx := &Transaction{
			From:   fcn.Addresses[0],
			To:     StorageContractAddress,
			Method: "addAsk",
			Params: []interface{}{miner, a},
		}

		if err := fcn.SendNewTransaction(tx); err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
	},
}

var OrderAskListCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		asks, err := listAsks(req.Context, fcn)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		re.Emit(asks)
	},
	Type: []*Ask{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			asks, ok := v.(*[]*Ask)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			for _, a := range *asks {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", a.MinerID, a.Size, a.Price, a.Expiry)
			}
			return nil
		}),
	},
}

func listAsks(ctx context.Context, fcn *FilecoinNode) ([]*Ask, error) {
	stroot := fcn.stateMgr.stateRoot
	sact, err := stroot.GetActor(ctx, StorageContractAddress)
	if err != nil {
		return nil, err
	}

	c, err := stroot.GetContract(ctx, sact.Code)
	if err != nil {
		return nil, err
	}

	cst, err := stroot.LoadContractState(ctx, sact.Memory)
	if err != nil {
		return nil, err
	}

	sc, ok := c.(*StorageContract)
	if !ok {
		return nil, fmt.Errorf("was not actually a storage contract somehow")
	}

	cctx := &CallContext{ContractState: cst, Ctx: ctx}

	ids, err := sc.loadArray(cctx, asksArrKey)
	if err != nil {
		return nil, err
	}
	var asks []*Ask
	for _, id := range ids {
		data, err := cst.Get(ctx, "a"+fmt.Sprint(id))
		if err != nil {
			return nil, err
		}
		var a Ask
		if err := json.Unmarshal(data, &a); err != nil {
			return nil, err
		}
		asks = append(asks, &a)
	}
	return asks, nil
}

func listBids(ctx context.Context, fcn *FilecoinNode) ([]*Bid, error) {
	stroot := fcn.stateMgr.stateRoot
	sact, err := stroot.GetActor(ctx, StorageContractAddress)
	if err != nil {
		return nil, err
	}

	c, err := stroot.GetContract(ctx, sact.Code)
	if err != nil {
		return nil, err
	}

	cst, err := stroot.LoadContractState(ctx, sact.Memory)
	if err != nil {
		return nil, err
	}

	sc, ok := c.(*StorageContract)
	if !ok {
		return nil, fmt.Errorf("was not actually a storage contract somehow")
	}

	cctx := &CallContext{ContractState: cst, Ctx: ctx}

	ids, err := sc.loadArray(cctx, bidsArrKey)
	if err != nil {
		return nil, err
	}
	var bids []*Bid
	for _, id := range ids {
		data, err := cst.Get(ctx, "b"+fmt.Sprint(id))
		if err != nil {
			return nil, err
		}
		var b Bid
		if err := json.Unmarshal(data, &b); err != nil {
			return nil, err
		}
		bids = append(bids, &b)
	}
	return bids, nil
}
