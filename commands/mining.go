package commands

import (
	"context"
	"fmt"
	"io"

	logging "gx/ipfs/QmPuosXfnE2Xrdiw95D78AhW41GYwGqpstKMf4TEsE4f33/go-log"
	cmds "gx/ipfs/QmUf5GFfV2Be3UtSAPKDVkoRd1TwEBTmx9TSSCFGGjNgdQ/go-ipfs-cmds"
	"gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	cmdkit "gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"

	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/mining"
	"github.com/filecoin-project/go-filecoin/state"
)

var log = logging.Logger("cmd/mining")

var miningCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Manage mining operations",
	},
	Subcommands: map[string]*cmds.Command{
		"once":  miningOnceCmd,
		"start": miningStartCmd,
		"stop":  miningStopCmd,
	},
}

var miningOnceCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) (err error) {
		req.Context = log.Start(req.Context, "miningOnceCmd")
		defer func() {
			log.SetTags(req.Context, map[string]interface{}{
				"args": req.Arguments,
				"path": req.Path,
			})
			log.FinishWithErr(req.Context, err)
		}()

		fcn := GetNode(env)

		cur := fcn.ChainMgr.GetBestBlock()
		log.LogKV(req.Context, "best-block", cur.Cid().String())

		addrs := fcn.Wallet.Addresses()
		if len(addrs) == 0 {
			return ErrNoWalletAddresses
		}
		rewardAddr := addrs[0]
		log.LogKV(req.Context, "reward-address", rewardAddr.String())

		blockGenerator := mining.NewBlockGenerator(fcn.MsgPool, func(ctx context.Context, cid *cid.Cid) (state.Tree, error) {
			return state.LoadStateTree(ctx, fcn.CborStore, cid, builtin.Actors)
		}, mining.ApplyMessages)
		// TODO(EC): Need to read best tipsets from storage and pass in. See also Node::StartMining().
		tipSets := []core.TipSet{{cur.Cid().String(): cur}}
		res := mining.MineOnce(req.Context, mining.NewWorker(blockGenerator), tipSets, rewardAddr)
		if res.Err != nil {
			return res.Err
		}
		if err := fcn.AddNewBlock(req.Context, res.NewBlock); err != nil {
			return err
		}
		log.LogKV(req.Context, "new-block", res.NewBlock.Cid().String())
		re.Emit(res.NewBlock.Cid()) // nolint: errcheck

		return nil
	},
	Type: cid.Cid{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, c *cid.Cid) error {
			fmt.Fprintln(w, c)
			return nil
		}),
	},
}

var miningStartCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) (err error) {
		req.Context = log.Start(req.Context, "miningStartCmd")
		defer func() {
			log.SetTags(req.Context, map[string]interface{}{
				"args": req.Arguments,
				"path": req.Path,
			})
			log.FinishWithErr(req.Context, err)
		}()

		if err := GetNode(env).StartMining(req.Context); err != nil {
			return err
		}
		re.Emit("Started mining\n") // nolint: errcheck

		return nil
	},
}

var miningStopCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) (err error) {
		req.Context = log.Start(req.Context, "miningStopCmd")
		defer func() {
			log.SetTags(req.Context, map[string]interface{}{
				"args": req.Arguments,
				"path": req.Path,
			})
			log.FinishWithErr(req.Context, err)
		}()

		GetNode(env).StopMining(req.Context)
		re.Emit("Stopped mining\n") // nolint: errcheck

		return nil
	},
}
