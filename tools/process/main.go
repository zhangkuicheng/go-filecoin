package main

import (
	"context"
	"io/ioutil"

	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("process")

func init() {
	logging.SetAllLoggers(4)
}

func main() {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "ACTION")
	if err != nil {
		panic(err)
	}
	env, err := NewEnvironment(dir)
	if err != nil {
		panic(err)
	}

	if err := env.AddGenesisMiner(ctx); err != nil {
		panic(err)
	}

	if err := env.AddNode(ctx); err != nil {
		panic(err)
	}

	if err := env.AddMiner(ctx); err != nil {
		panic(err)
	}
}
