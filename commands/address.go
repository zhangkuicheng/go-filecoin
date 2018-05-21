package commands

import (
	"fmt"
	"io"

	"gx/ipfs/QmUf5GFfV2Be3UtSAPKDVkoRd1TwEBTmx9TSSCFGGjNgdQ/go-ipfs-cmds"
	"gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
	"gx/ipfs/QmexBtiTTEwwn42Yi6ouKt6VqzpA6wjJgiW1oh9VfaRrup/go-multibase"

	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
)

var walletCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Manage your filecoin wallets",
	},
	Subcommands: map[string]*cmds.Command{
		"addrs":   addrsCmd,
		"balance": balanceCmd,
		"sign":    signCmd,
		"verify":  verifyCmd,
	},
}

var addrsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with addresses",
	},
	Subcommands: map[string]*cmds.Command{
		"ls":     addrsLsCmd,
		"new":    addrsNewCmd,
		"lookup": addrsLookupCmd,
	},
}

type addressResult struct {
	Address string
}

var addrsNewCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		fcn := GetNode(env)
		addr, err := fcn.NewAddress()
		if err != nil {
			return err
		}
		re.Emit(&addressResult{addr.String()}) // nolint: errcheck
		return nil
	},
	Type: &addressResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, a *addressResult) error {
			_, err := fmt.Fprintln(w, a.Address)
			return err
		}),
	},
}

var addrsLsCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		fcn := GetNode(env)
		for _, a := range fcn.Wallet.Addresses() {
			re.Emit(&addressResult{a.String()}) // nolint: errcheck
		}
		return nil
	},
	Type: &addressResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, addr *addressResult) error {
			_, err := fmt.Fprintln(w, addr.Address)
			return err
		}),
	},
}

var addrsLookupCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, false, "address to find peerId for"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		fcn := GetNode(env)

		address, err := types.NewAddressFromString(req.Arguments[0])
		if err != nil {
			return err
		}

		v, err := fcn.Lookup.Lookup(req.Context, address)
		if err != nil {
			return err
		}
		re.Emit(v.Pretty()) // nolint: errcheck
		return nil
	},
	Type: string(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, pid string) error {
			_, err := fmt.Fprintln(w, pid)
			return err
		}),
	},
}
var balanceCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, false, "address to get balance for"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		fcn := GetNode(env)
		blk := fcn.ChainMgr.GetBestBlock()
		if blk.StateRoot == nil {
			return ErrLatestBlockStateRootNil
		}

		tree, err := state.LoadStateTree(req.Context, fcn.CborStore, blk.StateRoot, builtin.Actors)
		if err != nil {
			return err
		}

		addr, err := types.NewAddressFromString(req.Arguments[0])
		if err != nil {
			return err
		}

		act, err := tree.GetActor(req.Context, addr)
		if err != nil {
			if state.IsActorNotFoundError(err) {
				// if the account doesn't exit, the balance should be zero
				re.Emit(types.NewTokenAmount(0)) // nolint: errcheck
				return nil
			}
			return err
		}

		re.Emit(act.Balance) // nolint: errcheck
		return nil
	},
	Type: &types.TokenAmount{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, b *types.TokenAmount) error {
			return PrintString(w, b)
		}),
	},
}

const reqEnc = multibase.Base32hex

// requireArgEncoding returns an error if `arg` is not encoded in `e`, or if
// decoding `arg` fails.
func requireArgEncoding(e multibase.Encoding, arg string) ([]byte, error) {
	enc, out, err := multibase.Decode(arg)
	if err != nil {
		return nil, err
	}
	if enc != e {
		return nil, fmt.Errorf("Encoding must be base32hex")
	}
	return out, nil
}

var signCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "sign data with address",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, false, "address to use for signing"),
		cmdkit.StringArg("data", true, false, "data to sign in base32hex encoding"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		fcn := GetNode(env)
		addr, err := types.NewAddressFromString(req.Arguments[0])
		if err != nil {
			return err
		}

		data, err := requireArgEncoding(reqEnc, req.Arguments[1])
		if err != nil {
			return err
		}

		sig, err := fcn.Wallet.Sign(addr, data)
		if err != nil {
			return err
		}

		out, err := multibase.Encode(reqEnc, sig)
		if err != nil {
			return err
		}

		re.Emit(out) // nolint: errcheck
		return nil

	},
	Type: string(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, sig string) error {
			_, err := fmt.Fprintf(w, "%s", sig)
			return err
		}),
	},
}

var verifyCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "verify signature of data against pubkey",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("pubkey", true, false, "public key to use for verification in base32hex encoding"),
		cmdkit.StringArg("data", true, false, "data to verify in base32hex encoding"),
		cmdkit.StringArg("sig", true, false, "sig to verify against in base32hex encoding"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		fcn := GetNode(env)

		// TODO: UX around this is poor, as it is not easy for a use to get,
		// a public key.
		bpub, err := requireArgEncoding(reqEnc, req.Arguments[0])
		if err != nil {
			return err
		}
		data, err := requireArgEncoding(reqEnc, req.Arguments[1])
		if err != nil {
			return err
		}
		sig, err := requireArgEncoding(reqEnc, req.Arguments[2])
		if err != nil {
			return err
		}

		valid, err := fcn.Wallet.Verify(bpub, data, sig)
		if err != nil {
			return err
		}

		re.Emit(valid) // nolint: errcheck
		return nil
	},
	Type: string(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, valid string) error {
			_, err := fmt.Fprintln(w, valid)
			return err
		}),
	},
}
