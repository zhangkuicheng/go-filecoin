package main

import (
	"os"

	//holdlogging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	logging "gx/ipfs/QmekXSLDnB9iTHRsKsidP6oN89vGGGRN27JP6gz9PSNHzR/go-log"

	"github.com/filecoin-project/go-filecoin/commands"
)

func main() {
	// TODO: make configurable - this should be done via a command like go-ipfs
	// something like:
	//		`go-filecoin log level "system" "level"`
	// TODO: find a better home for this
	// TODO fix this in go-log 4 == INFO
	// TODO(frrist) I don't want to deal with gx, and this is a hack so removing till
	// after demos
	/*
		n, err := strconv.Atoi(os.Getenv("GO_FILECOIN_LOG_LEVEL"))
		if err != nil {
			n = 4
		}
	*/

	//logging.SetAllLoggers(Level(n))
	logging.SetAllLoggers(4)

	// TODO implement help text like so:
	// https://github.com/ipfs/go-ipfs/blob/master/core/commands/root.go#L91
	// TODO don't panic if run without a command.
	code, _ := commands.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
