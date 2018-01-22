package commands

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	ma "gx/ipfs/QmW8s4zTsUoX1Q6CeYxVKPyqSKbF7H1YDUyTostBtZ8DaG/go-multiaddr"
	//peer "gx/ipfs/QmWNY7dV54ZDYmTA1ykVdwNCqC11mpU4zSUp6XDpLTH9eG/go-libp2p-peer"
	pstore "gx/ipfs/QmYijbtjCxFEjSXaudaQAUz3LN5VKLssm8WCUsRoqzXmQR/go-libp2p-peerstore"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	ipfsaddr "gx/ipfs/QmdMeXVB1V1SAZcFzoCuM3zR9K8PeuzCYg4zXNHcHh6dHU/go-ipfs-addr"
	"gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"

	contract "github.com/filecoin-project/playground/go-filecoin/contract"
	core "github.com/filecoin-project/playground/go-filecoin/core"
	libp2p "github.com/filecoin-project/playground/go-filecoin/libp2p"
	types "github.com/filecoin-project/playground/go-filecoin/types"

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
	cmdhttp "github.com/ipfs/go-ipfs-cmds/http"
)

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
	api := req.Options["api"].(string)

	hsh := fnv.New64()
	hsh.Write([]byte(api))
	seed := hsh.Sum64()

	r := rand.New(rand.NewSource(int64(seed)))
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	p2pcfg := libp2p.DefaultConfig()
	p2pcfg.PeerKey = priv

	// set up networking
	h, err := libp2p.Construct(context.Background(), p2pcfg)
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
	fcn, err := core.NewFilecoinNode(h, fsub, dag, bserv, bswap.(*bitswap.Bitswap))
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

	go func() {
		panic(http.ListenAndServe(api, handler))
	}()

	<-ch
	removeDaemonLock()
}

type CommandEnv struct {
	ctx  context.Context
	Node *core.FilecoinNode
}

func (ce *CommandEnv) Context() context.Context {
	return ce.ctx
}

func GetNode(env cmds.Environment) *core.FilecoinNode {
	ce := env.(*CommandEnv)
	return ce.Node
}

var AddrsCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"new":    AddrsNewCmd,
		"list":   AddrsListCmd,
		"lookup": AddrsLookupCmd,
	},
}

var AddrsNewCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)
		naddr := core.CreateNewAddress()
		fcn.Addresses = append(fcn.Addresses, naddr)
		re.Emit(naddr)
	},
	Type: types.Address(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			a, ok := v.(*types.Address)
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
	Type: []types.Address{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			addrs, ok := v.(*[]types.Address)
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

var AddrsLookupCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, false, "address to find peerID for"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)

		address, err := types.ParseAddress(req.Arguments[0])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		v, err := fcn.Lookup.Lookup(address)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		re.Emit(v.Pretty())
	},
	Type: string(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			pid, ok := v.(string)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			_, err := fmt.Fprintln(w, pid)
			if err != nil {
				return err
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
		re.Emit(fcn.Bitswap.GetWantlist())
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

		res := path.NewBasicResolver(fcn.DAG)

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
			a, err := types.ParseAddress(req.Arguments[0])
			if err != nil {
				re.SetError(err, cmdkit.ErrNormal)
				return
			}
			addr = a
		}

		stroot := fcn.StateMgr.GetStateRoot()
		act, err := stroot.GetActor(req.Context, contract.FilecoinContractAddr)
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

		cctx := &contract.CallContext{Ctx: req.Context, ContractState: cst}
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
		toaddr, err := types.ParseAddress(req.Arguments[1])
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		from := fcn.Addresses[0]

		nonce, err := fcn.StateMgr.GetStateRoot().NonceForActor(req.Context, from)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		tx := &types.Transaction{
			From:   from,
			To:     contract.FilecoinContractAddr,
			Nonce:  nonce,
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
	Type: types.Address(""),
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		fcn := GetNode(env)

		pledge, ok := big.NewInt(0).SetString(req.Arguments[0], 10)
		if !ok {
			re.SetError("failed to parse pledge as number", cmdkit.ErrNormal)
			return
		}

		from := fcn.Addresses[0]

		nonce, err := fcn.StateMgr.StateRoot.NonceForActor(req.Context, from)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		tx := &types.Transaction{
			From:   fcn.Addresses[0],
			To:     contract.StorageContractAddress,
			Nonce:  nonce,
			Method: "createMiner",
			Params: []interface{}{pledge},
		}

		res, err := fcn.SendNewTransactionAndWait(req.Context, tx)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		if !res.Receipt.Success {
			re.SetError("miner creation failed", cmdkit.ErrNormal)
			return
		}

		resStr, ok := res.Receipt.Result.(types.Address)
		if !ok {
			re.SetError("createMiner call didn't return an address", cmdkit.ErrNormal)
			return
		}

		re.Emit(resStr)
	},
}

var OrderCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"bid":  OrderBidCmd,
		"ask":  OrderAskCmd,
		"deal": OrderDealCmd,
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

		from := fcn.Addresses[0]

		nonce, err := fcn.StateMgr.StateRoot.NonceForActor(req.Context, from)

		tx := &types.Transaction{
			From:   fcn.Addresses[0],
			To:     contract.StorageContractAddress,
			Nonce:  nonce,
			Method: "addBid",
			Params: []interface{}{uint64(price), uint64(size)},
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
	Type: []*contract.Bid{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			bids, ok := v.(*[]*contract.Bid)
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
		miner, err := types.ParseAddress(req.Arguments[0])
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

		from := fcn.Addresses[0]

		nonce, err := fcn.StateMgr.StateRoot.NonceForActor(req.Context, from)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}

		tx := &types.Transaction{
			From:   fcn.Addresses[0],
			To:     contract.StorageContractAddress,
			Nonce:  nonce,
			Method: "addAsk",
			Params: []interface{}{miner, int64(price), uint64(size)},
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
	Type: []*contract.Ask{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			asks, ok := v.(*[]*contract.Ask)
			if !ok {
				return fmt.Errorf("unexpected type: %T", v)
			}

			for _, a := range *asks {
				fmt.Fprintf(w, "%s\t%d\t%s\t%d\n", a.MinerID, a.Size, a.Price, a.Expiry)
			}
			return nil
		}),
	},
}

func listAsks(ctx context.Context, fcn *core.FilecoinNode) ([]*contract.Ask, error) {
	stroot := fcn.StateMgr.GetStateRoot()
	sact, err := stroot.GetActor(ctx, contract.StorageContractAddress)
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

	sc, ok := c.(*contract.StorageContract)
	if !ok {
		return nil, fmt.Errorf("was not actually a storage contract somehow")
	}

	cctx := &contract.CallContext{ContractState: cst, Ctx: ctx}

	return sc.ListAsks(cctx)
}

func listBids(ctx context.Context, fcn *core.FilecoinNode) ([]*contract.Bid, error) {
	stroot := fcn.StateMgr.GetStateRoot()
	sact, err := stroot.GetActor(ctx, contract.StorageContractAddress)
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

	sc, ok := c.(*contract.StorageContract)
	if !ok {
		return nil, fmt.Errorf("was not actually a storage contract somehow")
	}

	cctx := &contract.CallContext{ContractState: cst, Ctx: ctx}
	return sc.ListBids(cctx)
}

var OrderDealCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"make": OrderDealMakeCmd,
	},
}

var OrderDealMakeCmd = &cmds.Command{
	Options: []cmdkit.Option{
		cmdkit.StringOption("mode", "mode to make deal in, client or miner"),
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("ask", true, false, "id of ask for deal"),
		cmdkit.StringArg("bid", true, false, "id of bid for deal"),
		cmdkit.StringArg("miner", false, false, "id of miner to make deal with"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		// TODO: right now, the flow is that this command gets called by the miner.
		// we still need to work out the process here...
		fcn := GetNode(env)

		from := fcn.Addresses[0]

		ask := req.Arguments[0]
		bid := req.Arguments[1]

		var miner string
		if len(req.Arguments) > 2 {
			miner = req.Arguments[2]
		}

		makesignature := func(who string) string {
			// TODO: crypto...
			return who
		}

		sig := makesignature(miner)

		nonce, err := fcn.StateMgr.GetStateRoot().NonceForActor(req.Context, from)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		tx := &types.Transaction{
			From:   from,
			To:     contract.StorageContractAddress,
			Nonce:  nonce,
			Method: "makeDeal",
			Params: []interface{}{ask, bid, sig},
		}

		fcn.SendNewTransaction(tx)
	},
	Type: []*contract.Ask{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error {
			return nil
		}),
	},
}
