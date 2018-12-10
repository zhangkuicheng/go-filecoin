package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	iptb "github.com/ipfs/iptb/testbed"

	"github.com/filecoin-project/go-filecoin/address"
	gengen "github.com/filecoin-project/go-filecoin/gengen/util"
	"github.com/filecoin-project/go-filecoin/types"
)

// GenesisInfo chains require information to start a single node with funds
type GenesisInfo struct {
	Path          string
	KeyFile       string
	WalletAddress address.Address
	MinerAddress  address.Address
}

type idResult struct {
	ID string
}

func GenerateGenesis(funds int64, dir string) (*GenesisInfo, error) {
	// Setup, generate a genesis and key file
	cfg := &gengen.GenesisCfg{
		Keys: 1,
		PreAlloc: []string{
			strconv.FormatInt(funds, 10),
		},
		Miners: []gengen.Miner{
			{
				Owner: 0,
				Power: 1,
			},
		},
	}

	genfile, err := ioutil.TempFile(dir, "genesis.*.car")
	if err != nil {
		return nil, err
	}

	keyfile, err := ioutil.TempFile(dir, "wallet.*.key")
	if err != nil {
		return nil, err
	}

	info, err := gengen.GenGenesisCar(cfg, genfile, 0)
	if err != nil {
		return nil, err
	}

	key := info.Keys[0]
	if err := json.NewEncoder(keyfile).Encode(key); err != nil {
		return nil, err
	}

	walletAddr, err := key.Address()
	if err != nil {
		return nil, err
	}

	minerAddr := info.Miners[0].Address

	return &GenesisInfo{
		Path:          genfile.Name(),
		KeyFile:       keyfile.Name(),
		WalletAddress: walletAddr,
		MinerAddress:  minerAddr,
	}, nil
}

type Environment struct {
	Location     string // ideally something more general than a path
	GenesisFile  *GenesisInfo
	GenesisMiner *Filecoin

	processes []*Filecoin
}

func NewEnvironment(dir string) (*Environment, error) {
	log.Infof("Creating Environment: %s", dir)
	gf, err := GenerateGenesis(10000000, dir)
	if err != nil {
		return nil, err
	}

	return &Environment{
		Location:    dir,
		GenesisFile: gf,
		processes:   nil,
	}, nil
}

func (e *Environment) AddProcess(p *Filecoin) {
	for _, n := range e.processes {
		n.Connect(context.Background(), p)
	}
	e.processes = append(e.processes, p)
}

func (e *Environment) Processes(p *Filecoin) []*Filecoin {
	return e.processes
}

// TODO don't put "go-filecoin" in the path, tell the process what to use here
func (e *Environment) NewProcess(ctx context.Context, t, d string) (*Filecoin, error) {
	log.Infof("NewProcess, type: %s, dir: %s", t, d)
	ns := iptb.NodeSpec{
		Type: t,
		Dir:  d,
	}

	c, err := ns.Load()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(d, 0775); err != nil {
		return nil, err
	}

	return &Filecoin{
		Core: c,

		pluginType: t,
		pluginDir:  d,

		ctx: ctx,
	}, nil
}

func (e *Environment) CreateGenesisMiner(ctx context.Context) (*Filecoin, error) {
	log.Info("Run CreateGenesisMiner")
	// Create a filecoin process structure
	fc, err := e.NewProcess(ctx, "localfilecoin", fmt.Sprintf("%s/0", e.Location))
	if err != nil {
		return nil, err
	}

	// init the filecoin node with a genesis file
	_, err = fc.Init(ctx, "--genesisfile", e.GenesisFile.Path)
	if err != nil {
		return nil, err
	}

	// start the filecoin node
	err = fc.DaemonStart()
	if err != nil {
		return nil, err
	}

	// add the miner address to the nodes config
	_, err = fc.RunCmd(ctx, nil, "go-filecoin", "config", "mining.minerAddress", e.GenesisFile.MinerAddress.String())
	if err != nil {
		return nil, err
	}

	// import the miner address key into the nodes wallet
	_, err = fc.RunCmd(ctx, nil, "go-filecoin", "wallet", "import", e.GenesisFile.KeyFile)
	if err != nil {
		return nil, err
	}

	// update the nodes peerID to own said miner
	_, err = fc.UpdatePeerID(e.GenesisFile.WalletAddress, e.GenesisFile.MinerAddress, fc.ID)
	if err != nil {
		return nil, err
	}

	fc.DefaultWalletAddr = e.GenesisFile.WalletAddress
	fc.MinerOwner = e.GenesisFile.WalletAddress
	fc.MinerAddress = e.GenesisFile.MinerAddress

	if err := fc.MiningStart(); err != nil {
		return nil, err
	}

	return fc, nil
}

func (e *Environment) AddGenesisMiner(ctx context.Context) error {
	fc, err := e.CreateGenesisMiner(ctx)
	if err != nil {
		return err
	}
	e.GenesisMiner = fc
	e.AddProcess(fc)
	return nil
}

func (e *Environment) CreateNode(ctx context.Context) (*Filecoin, error) {
	// Create a filecoin process structure
	fc, err := e.NewProcess(ctx, "localfilecoin", fmt.Sprintf("%s/%d", e.Location, len(e.processes)))
	if err != nil {
		return nil, err
	}

	// init the filecoin node with a genesis file
	_, err = fc.Init(ctx, "--genesisfile", e.GenesisFile.Path)
	if err != nil {
		return nil, err
	}

	// start the filecoin node
	err = fc.DaemonStart()
	if err != nil {
		return nil, err
	}

	addrs, err := fc.AddressLs()
	if err != nil {
		return nil, err
	}

	// sanity check
	if len(addrs) != 1 {
		panic(addrs)
	}
	fc.DefaultWalletAddr, err = address.NewFromString(addrs[0])
	if err != nil {
		return nil, err
	}

	return fc, nil
}

func (e *Environment) AddNode(ctx context.Context) error {
	log.Info("Adding Node")
	fc, err := e.CreateNode(ctx)
	if err != nil {
		return err
	}
	e.AddProcess(fc)
	return nil
}

func (e *Environment) CreateMiner(ctx context.Context) (*Filecoin, error) {
	fc, err := e.CreateNode(ctx)
	if err != nil {
		return nil, err
	}
	e.AddProcess(fc)

	value, ok := types.NewAttoFILFromFILString("100")
	if !ok {
		panic("here")
	}

	_, err = e.GenesisMiner.SendFilecoin(e.GenesisMiner.DefaultWalletAddr, fc.DefaultWalletAddr, value)
	if err != nil {
		return nil, err
	}

	_, err = e.GenesisMiner.MiningOnce()
	if err != nil {
		return nil, err
	}

	// TODO should probabaly wait on message and check receipt
	fc.MinerAddress, err = fc.CreateMiner(fc.DefaultWalletAddr, 100, fc.ID, value)
	if err != nil {
		return nil, err
	}
	fc.MinerOwner = fc.DefaultWalletAddr

	return fc, nil
}

func (e *Environment) AddMiner(ctx context.Context) error {
	miner, err := e.CreateMiner(ctx)
	if err != nil {
		return err
	}
	e.AddProcess(miner)
	return nil
}
