package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"gx/ipfs/QmQZadYTDF4ud9DdK85PH2vReJRzUM9YfVW4ReB1q2m51p/go-hamt-ipld"
	bserv "gx/ipfs/QmUSuYd5Q1N291DH679AVvHwGLwtS1V9VPDWvnUN9nGJPT/go-blockservice"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmWdao8WJqYU65ZbYQyQWMFqku6QFxkPiv8HSUAkXdHZoe/go-ipfs-exchange-offline"
	"gx/ipfs/QmZFbDTY9jfSBms2MchvYM9oYRbAF19K7Pby47yDBfpPrb/go-cid"
	bstore "gx/ipfs/QmcmpX42gtDv1fz24kau4wjS9hfwWj5VexWBKgGnWzsyag/go-ipfs-blockstore"
	"gx/ipfs/QmVG5gxteQNEMhrS8prJSmU2C9rebtFuTd3SYZ5kE3YZ5k/go-datastore"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/chain"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/mining"
	"github.com/filecoin-project/go-filecoin/node"
	"github.com/filecoin-project/go-filecoin/repo"
	"github.com/filecoin-project/go-filecoin/state"
	th "github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/filecoin-project/go-filecoin/types"
)

var genesisKey = datastore.NewKey("/consensus/genesisCid")
var length int
var binom bool
var repodir string

func init() {
	flag.IntVar(&length, "length", 5, "length of fake chain to create")

	// Default repodir is different than Filecoin to avoid accidental clobbering of real data.
	flag.StringVar(&repodir, "repodir", "~/.fakecoin", "repo directory to use")

	flag.BoolVar(&binom, "binom", false, "generate multiblock tipsets where the number of blocks per epoch is drawn from a a realistic distribution")
}

func build(ctx context.Context, r repo.Repo) (chain.Store, bstore.Blockstore, *hamt.CborIpldStore, consensus.Protocol, error) {
	bs := bstore.NewBlockstore(r.Datastore())
	cst := &hamt.CborIpldStore{Blocks: bserv.New(bs, offline.Exchange(bs))}
	// see if this repo has been initialized with a genesis hash
	genCid, err := readGenesisCid(r.Datastore())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var chainStore chain.Store
	if err != nil {
		// Initialize the datastore with the default gif
		genesis, err := consensus.InitGenesis(cst, bs)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		genTS, err := consensus.NewTipSet(genesis)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		genCid = genesis.Cid()
		chainStore = chain.NewDefaultStore(r.ChainDatastore(), cst, genCid)
		defer chainStore.Stop()
		chainStore.PutTipSetAndState(ctx, &chain.TipSetAndState{
			TipSet:          genTS,
			TipSetStateRoot: genesis.StateRoot,
		})
	} else {
		chainStore = chain.NewDefaultStore(r.ChainDatastore(), cst, genCid)
		defer chainStore.Stop()
	}
	err = chainStore.Load(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	con := consensus.NewExpected(cst, bs, &consensus.TestView{}, genCid)
	return chainStore, bs, cst, con, nil
}

func main() {
	ctx := context.Background()

	var cmd string

	if len(os.Args) > 1 {
		cmd = os.Args[1]
		if len(os.Args) > 2 {
			// Remove the cmd argument if there are options, to satisfy flag.Parse() while still allowing a command-first syntax.
			os.Args = append(os.Args[1:], os.Args[0])
		}
	}
	flag.Parse()

	// Initialize chain.Store from repo.
	r, err := repo.OpenFSRepo(repodir)
	if err != nil {
		log.Fatal(err)
	}
	defer closeRepo(r)
	chainStore, bs, cst, con, err := build(ctx, r)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error building stores from repo"))
	}

	switch cmd {
	default:
		flag.Usage()
	case "fake":
		if err = cmdfake(ctx, length, binom, chainStore); err != nil {
			log.Fatal(err)
		}
	case "actors":
		if err := cmdFakeActors(ctx, chainStore, con, bs, cst); err != nil {
			log.Fatal(err)
		}
<<<<<<< HEAD
=======
			log.Fatal(err)
		}
	}
}

func closeRepo(r *repo.FSRepo) {
	err := r.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func getWorker(msgPool *core.MessagePool, chainStore chain.Store, con consensus.Protocol, cst *hamt.CborIpldStore, bs bstore.Blockstore) *mining.DefaultWorker {
	ma := types.MakeTestAddress("miningAddress")
	getStateFromKey := func(ctx context.Context, tsKey string) (state.Tree, error) {
		tsas, err := chainStore.GetTipSetAndState(ctx, tsKey)
		if err != nil {
			return nil, err
		}
		return state.LoadStateTree(ctx, cst, tsas.TipSetStateRoot, builtin.Actors)
	}
	getState := func(ctx context.Context, ts consensus.TipSet) (state.Tree, error) {
		return getStateFromKey(ctx, ts.String())
	}
	getWeight := func(ctx context.Context, ts consensus.TipSet) (uint64, uint64, error) {
		parent, err := ts.Parents()
		if err != nil {
			return uint64(0), uint64(0), err
		}
		// TODO handle genesis cid more gracefully
		if parent.Len() == 0 {
			return con.Weight(ctx, ts, nil)
		}
		pSt, err := getStateFromKey(ctx, parent.String())
		if err != nil {
			return uint64(0), uint64(0), err
		}
		return con.Weight(ctx, ts, pSt)
	}

	return mining.NewDefaultWorker(msgPool, getState, getWeight,
		consensus.ApplyMessages, &consensus.TestView{}, bs, cst, ma,
		mining.BlockTimeTest)
}

func fake(ctx context.Context, length int, binom bool, chainStore chain.Store) error {
	ts := chainStore.Head()
	if ts == nil {
		return errors.New("head of chain unset")
	}
	// If a binomial distribution is specified we generate a binomially
	// distributed number of blocks per epoch
	/*	if binom {
		_, err := core.AddChainBinomBlocksPerEpoch(ctx, processNewBlock, stateFromTS, ts, 100, length)
		if err != nil {
			return err
		}
		fmt.Printf("Added chain of %d empty epochs.\n", length)
		return err
	}*/
	// The default block distribution just adds a linear chain of 1 block
	// per epoch.
	_, err := chain.AddChain(ctx, chainStore, ts.ToSlice(), length)
	if err != nil {
		return err
	}
	fmt.Printf("Added chain of %d empty blocks.\n", length)

	return err
}

// fakeActors adds a block ensuring that the StateTree contains at least one of each extant Actor type, along with
// well-formed data in its memory. For now, this exists primarily to exercise the Filecoin Explorer, though it may
// be used for testing in the future.
func fakeActors(ctx context.Context, cst *hamt.CborIpldStore, chainStore chain.Store, con consensus.Protocol, bs bstore.Blockstore, bts consensus.TipSet) (err error) {
	messageWaiter := node.NewMessageWaiter(chainStore, bs, cst)
	msgPool := core.NewMessagePool()

	//// Have the storage market actor create a new miner
	params, err := abi.ToEncodedValues(types.NewBytesAmount(100000), []byte{}, th.RequireRandomPeerID())
	if err != nil {
		return err
	}

	// TODO address support for signed messages
	newMinerMessage := types.NewMessage(address.TestAddress, address.StorageMarketAddress, 0, types.NewAttoFILFromFIL(400), "createMiner", params)
	newSingedMinerMessage, err := types.NewSignedMessage(*newMinerMessage, nil)
	if err != nil {
		return err
	}
	_, err = msgPool.Add(newSingedMinerMessage)
	if err != nil {
		return err
	}

	blk, err := mineBlock(ctx, msgPool, cst, chainStore, con, bs, bts.ToSlice())
	if err != nil {
		return err
	}
	msgPool = core.NewMessagePool()

	cid, err := newMinerMessage.Cid()
	if err != nil {
		return err
	}

	var createMinerReceipt *types.MessageReceipt
	err = messageWaiter.WaitForMessage(ctx, cid, func(b *types.Block, msg *types.SignedMessage, rcp *types.MessageReceipt) error {
		createMinerReceipt = rcp
		return nil
	})
	if err != nil {
		return err
	}

	minerAddress, err := types.NewAddressFromBytes(createMinerReceipt.Return[0])
	if err != nil {
		return err
	}

	// Add a new ask to the storage market
	params, err = abi.ToEncodedValues(types.NewAttoFILFromFIL(10), types.NewBytesAmount(1000))
	if err != nil {
		return err
	}
	// TODO address support for signed messages
	askMsg := types.NewMessage(address.TestAddress, minerAddress, 1, types.NewAttoFILFromFIL(100), "addAsk", params)
	askSignedMessage, err := types.NewSignedMessage(*askMsg, nil)
	if err != nil {
		return err
	}
	_, err = msgPool.Add(askSignedMessage)
	if err != nil {
		return err
	}

	// Add a new bid to the storage market
	params, err = abi.ToEncodedValues(types.NewAttoFILFromFIL(9), types.NewBytesAmount(10))
	if err != nil {
		return err
	}
	// TODO address support for signed messages
	bidMsg := types.NewMessage(address.TestAddress2, address.StorageMarketAddress, 0, types.NewAttoFILFromFIL(90), "addBid", params)
	bidSignedMessage, err := types.NewSignedMessage(*bidMsg, nil)
	if err != nil {
		return err
	}
	_, err = msgPool.Add(bidSignedMessage)
	if err != nil {
		return err
	}

	// mine again
	blk, err = mineBlock(ctx, msgPool, cst, chainStore, con, bs, []*types.Block{blk})
	if err != nil {
		return err
	}
	msgPool = core.NewMessagePool()

	// Create deal
	params, err = abi.ToEncodedValues(big.NewInt(0), big.NewInt(0), address.TestAddress2, types.NewCidForTestGetter()().Bytes())
	if err != nil {
		return err
	}
	// TODO address support for signed messages
	newDealMessage := types.NewMessage(address.TestAddress, address.StorageMarketAddress, 2, types.NewAttoFILFromFIL(400), "addDeal", params)
	newDealSignedMessage, err := types.NewSignedMessage(*newDealMessage, nil)
	if err != nil {
		return err
	}
	_, err = msgPool.Add(newDealSignedMessage)
	if err != nil {
		return err
	}

	_, err = mineBlock(ctx, msgPool, cst, chainStore, con, bs, []*types.Block{blk})
	return err
}

func mineBlock(ctx context.Context, mp *core.MessagePool, cst *hamt.CborIpldStore, chainStore chain.Store, con consensus.Protocol, bs bstore.Blockstore, blks []*types.Block) (*types.Block, error) {
	mw := getWorker(mp, chainStore, con, cst, bs)

	const nullBlockCount = 0
	ts, err := consensus.NewTipSet(blks...)
	if err != nil {
		return nil, err
	}
	blk, err := mw.Generate(ctx, ts, nil, nullBlockCount)

	if err != nil {
		return nil, err
	}
	newTS, err := consensus.NewTipSet(blk)
	if err != nil {
		return nil, err
	}
	err = chainStore.PutTipSetAndState(ctx, &chain.TipSetAndState{
		TipSet:          newTS,
		TipSetStateRoot: blk.StateRoot,
	})
	if err != nil {
		return nil, err
	}

	err = chainStore.SetHead(ctx, newTS)
	if err != nil {
		return nil, err
	}

	return blk, nil
>>>>>>> The chain refactor -- no commit message can tell of the horror
}

// readGenesisCid is a helper function that queries the provided datastore forr
// an entry with the genesisKey cid, returning if found.
func readGenesisCid(ds datastore.Datastore) (*cid.Cid, error) {
	bbi, err := ds.Get(genesisKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read genesisKey")
	}
	bb, ok := bbi.([]byte)
	if !ok {
		return nil, fmt.Errorf("stored genesisCid not []byte")
	}

	var c cid.Cid
	err = json.Unmarshal(bb, &c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to cast genesisCid")
	}
	return &c, nil
}
