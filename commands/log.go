package commands

import (
	"io"

	cmds "gx/ipfs/QmYMj156vnPY7pYvtkvQiMDAzqWDDHkfiW5bYbMpYoHxhB/go-ipfs-cmds"
	cmdkit "gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"

	// TODO This gx package isn't published yet
	writer "gx/ipfs/Qmaf59ke1Gu4rz9tP8MzCp6PyGv9ZU9cNJvPwrwNavSL9r/go-log/writer"
)

var logCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with the daemon log output.",
		ShortDescription: `
'ipfs log' contains utility commands to affect or read the logging
output of a running daemon.
`,
	},

	Subcommands: map[string]*cmds.Command{
		"tail": logTailCmd,
	},
}

var logTailCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Read the event log.",
		ShortDescription: `
Outputs event log messages (not other log messages) as they are generated.
`,
	},

	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		ctx := req.Context
		r, w := io.Pipe()
		go func() {
			defer w.Close()
			<-ctx.Done()
		}()
		writer.WriterGroup.AddWriter(w)
		return re.Emit(r)
	},
}
