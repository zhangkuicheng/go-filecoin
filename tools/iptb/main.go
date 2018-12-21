package main

import (
	"fmt"
	"os"

	cli "github.com/ipfs/iptb/cli"
	testbed "github.com/ipfs/iptb/testbed"

	docker "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/docker"
	local "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

func init() {
	// Super hacky?
	os.Setenv("IPTB_ROOT", "+")

	_, err := testbed.RegisterPlugin(testbed.IptbPlugin{
		From:       "<builtin>",
		NewNode:    local.NewNode,
		PluginName: local.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}

	_, err = testbed.RegisterPlugin(testbed.IptbPlugin{
		From:       "<builtin>",
		NewNode:    docker.NewNode,
		PluginName: docker.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}

	if err != nil {
		panic(err)
	}
}

func main() {
	cli := cli.NewCli()
	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintf(cli.ErrWriter, "%s\n", err)
		os.Exit(1)
	}
}
