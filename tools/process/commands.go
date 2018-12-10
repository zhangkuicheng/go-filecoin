package main

import (
	"fmt"
	"math/big"
	"strings"

	//"github.com/filecoin-project/go-filecoin/api"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	"gx/ipfs/QmZMWMvWMVKCbHetJ4RgndbuEF1io2UpUxwQwtNjtYPzSC/go-ipfs-files"
	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/protocol/storage"
	"github.com/filecoin-project/go-filecoin/types"
)

// Starts the daemon and cache its peerID
func (f *Filecoin) DaemonStart() error {
	log.Info("Starting Filecoin daemon")
	_, err := f.Start(f.ctx, false)
	if err != nil {
		return err
	}

	pid, err := f.PeerID()
	if err != nil {
		return err
	}
	log.Infof("Started Daemon with peerID: %s", pid)

	f.ID, err = peer.IDB58Decode(pid)
	if err != nil {
		panic(err)
		return err
	}
	return nil
}

func (f *Filecoin) CreateMiner(fromAddr address.Address, pledge uint64, pid peer.ID, collateral *types.AttoFIL) (address.Address, error) {
	var out address.Address
	s_fromAddr := fromAddr.String()
	s_pledge := fmt.Sprintf("%d", pledge)
	s_pid := pid.Pretty()
	s_collateral := collateral.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "create", s_pledge, s_collateral, "--from", s_fromAddr, "--peerid", s_pid); err != nil {
		return address.Address{}, err
	}

	return out, nil
}

func (f *Filecoin) UpdatePeerID(fromAddr, minerAddr address.Address, newPid peer.ID) (cid.Cid, error) {
	var out cid.Cid
	s_fromAddr := fromAddr.String()
	s_minerAddr := minerAddr.String()
	s_newPid := newPid.Pretty()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "update-peerid", s_minerAddr, s_newPid, "--from", s_fromAddr); err != nil {
		return cid.Undef, err
	}

	return out, nil
}

func (f *Filecoin) AddAsk(fromAddr, minerAddr address.Address, price *types.AttoFIL, expiry *big.Int) (cid.Cid, error) {
	var out cid.Cid
	s_fromAddr := fromAddr.String()
	s_minerAddr := minerAddr.String()
	s_price := price.String()
	s_expiry := expiry.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "add-ask", s_minerAddr, s_price, s_expiry, "--from", s_fromAddr); err != nil {
		return cid.Undef, err
	}
	return out, nil
}

func (f *Filecoin) GetOwner(minerAddr address.Address) (address.Address, error) {
	var out address.Address
	s_minerAdd := minerAddr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "owner", s_minerAdd); err != nil {
		return address.Address{}, err
	}

	return out, nil
}

/*****************************************************************************/
/****************************MINING*******************************************/

func (f *Filecoin) MiningOnce() (*types.Block, error) {
	var out types.Block

	if err := f.RunCmdJSON(&out, "go-filecoin", "mining", "once"); err != nil {
		return nil, err
	}
	return &out, nil
}

// TODO check exit code
func (f *Filecoin) MiningStart() error {
	_, err := f.RunCmd(f.ctx, nil, "go-filecoin", "mining", "start")
	if err != nil {
		return err
	}
	return nil
}

// TODO check exit code
func (f *Filecoin) MiningStop() error {
	_, err := f.RunCmd(f.ctx, nil, "go-filecoin", "mining", "stop")
	if err != nil {
		return err
	}
	return nil
}

/*****************************************************************************/
/****************************CLIENT*******************************************/

func (f *Filecoin) ProposeStorageDeal(data cid.Cid, miner address.Address, ask uint64, duration uint64) (*storage.DealResponse, error) {
	var out storage.DealResponse
	s_data := data.String()
	s_miner := miner.String()
	s_ask := fmt.Sprintf("%d", ask)
	s_duration := fmt.Sprintf("%d", duration)

	if err := f.RunCmdJSON(&out, "go-filecoin", "client", "propose-storage-deal", s_miner, s_data, s_ask, s_duration); err != nil {
		return nil, err
	}
	return &out, nil
}

func (f *Filecoin) QueryStorageDeal(prop cid.Cid) (*storage.DealResponse, error) {
	var out storage.DealResponse
	s_prop := prop.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "client", "query-storage-deal", s_prop); err != nil {
		return nil, err
	}
	return &out, nil
}

/*****************************************************************************/
/****************************WALLET*******************************************/

func (f *Filecoin) WalletBalance(addr address.Address) (*types.AttoFIL, error) {
	// TODO will probably break with json
	var out types.AttoFIL
	s_addr := addr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "wallet", "balance", s_addr); err != nil {
		return nil, err
	}
	return &out, nil
}

func (f *Filecoin) WalletImport(file files.File) ([]address.Address, error) {
	var out []address.Address
	s_file := file.FullPath()

	if err := f.RunCmdJSON(&out, "go-filecoin", "wallet", "import", s_file); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *Filecoin) WalletExport(addrs []address.Address) ([]*types.KeyInfo, error) {
	var out []*types.KeyInfo
	var s_addrs []string
	for _, a := range addrs {
		s_addrs = append(s_addrs, a.String())
	}

	if err := f.RunCmdJSON(&out, "go-filecoin", "wallet", "export", strings.Join(s_addrs, " ")); err != nil {
		return nil, err
	}

	return out, nil
}

/*****************************************************************************/
/****************************ADDRESS*******************************************/

func (f *Filecoin) AddressNew() (address.Address, error) {
	var out address.Address

	if err := f.RunCmdJSON(&out, "go-filecoin", "address", "new"); err != nil {
		return address.Address{}, err
	}
	return out, nil
}
func (f *Filecoin) AddressLs() ([]string, error) {
	var out []string

	if err := f.RunCmdJSON(&out, "go-filecoin", "address", "ls"); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *Filecoin) AddressLookup(addr address.Address) (peer.ID, error) {
	var out peer.ID
	s_addr := addr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "address", "lookup", s_addr); err != nil {
		return "", err
	}
	return out, nil
}

func (f *Filecoin) SendFilecoin(from, to address.Address, val *types.AttoFIL) (cid.Cid, error) {
	var out cid.Cid
	s_from := from.String()
	s_to := to.String()
	s_val := val.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "message", "send", s_to, "--from", s_from, "--value", s_val); err != nil {
		return cid.Undef, err
	}
	return out, nil
}

// TODO I think we may need to rething the types returned by these interface methods
// returning a channel or an interface is tough to deal with
/*
func (f *Filecoin) ListAsks(ctx context.Context) (<-chan Ask, error) {
}
func (f *Filecoin) Cat(c cid.Cid) (uio.DagReader, error) {
}
func (f *Filecoin) ImportData(data io.Reader) (ipld.Node, error) {
}

*/

// TODO not sure how this is going to work with bigints
/*
func (f *Filecoin) GetPledge(minerAddr address.Address) (*big.Int, error) {
	// TODO I don't think this is going to play nicely with json, just a gut feeling
	var out string
	s_minerAddr := minerAddr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "pledge", s_minerAddr); err != nil {
		return nil, err
	}

	return out, nil
}

func (f *Filecoin) GetPower(minerAddr address.Address) (*big.Int, error) {
	// TODO I don't think this is going to play nicely with json, just a gut feeling
	var put string
	s_minerAddr := minerAddr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "power", s_minerAddr); err != nil {
		return nil, err
	}

	return out, nil
}
*/
