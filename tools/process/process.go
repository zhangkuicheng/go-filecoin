package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	iptb "github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/address"
	dockerplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/docker"
	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

// must register all plugins
func init() {
	_, err := iptb.RegisterPlugin(iptb.IptbPlugin{
		From:       "<builtin>",
		NewNode:    localplugin.NewNode,
		PluginName: localplugin.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}

	_, err = iptb.RegisterPlugin(iptb.IptbPlugin{
		From:       "<builtin>",
		NewNode:    dockerplugin.NewNode,
		PluginName: dockerplugin.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}
}

type Filecoin struct {
	testbedi.Core //localfilecoin, dockerfilecoin, kubernetesfilecoin

	ID                peer.ID
	DefaultWalletAddr address.Address
	MinerAddress      address.Address
	MinerOwner        address.Address

	pluginType string
	pluginDir  string

	ctx context.Context
}

// TODO don't put "go-filecoin" in the path, tell the process what to use here
func NewProcess(ctx context.Context, t, d string) (*Filecoin, error) {
	ns := iptb.NodeSpec{
		Type: t,
		Dir:  d,
	}

	c, err := ns.Load()
	if err != nil {
		return nil, err
	}

	return &Filecoin{
		Core: c,

		pluginType: t,
		pluginDir:  d,

		ctx: ctx,
	}, nil
}

// MustRunCmdJSON runs `args` against TestNode. The '--enc=json' flag is appened to the command specified by `args`,
// the result of the command is marshaled into `expOut`.
func (f *Filecoin) RunCmdJSON(expOut interface{}, args ...string) error {
	args = append(args, "--enc=json")
	out, err := f.RunCmd(f.ctx, nil, args...)
	if err != nil {
		return err
	}
	// did the command exit with nonstandard exit code?
	if out.ExitCode() > 0 {
		return errors.New(fmt.Sprintf("Filecoin command: %s, exited with non-zero exitcode: %d", out.Args(), out.ExitCode()))
	}

	dec := json.NewDecoder(out.Stdout())
	if err := dec.Decode(expOut); err != nil {
		return err
	}
	return nil
}
