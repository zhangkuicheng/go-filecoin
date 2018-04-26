package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	iptb "github.com/filecoin-project/iptb/util"
	"github.com/stretchr/testify/require"
)

func TestMiningChain(t *testing.T) {
	require := require.New(t)
	numNodes := 50
	// Init
	cfg := &iptb.InitCfg{
		Count:      numNodes,
		NodeType:   "filecoin",
		DeployType: "local",
		Force:      true,
		Bootstrap:  "skip",
		//PortStart:  0,
		//Mdns:       false,
		//Utp:        false,
		//Websocket:  false,
		//Override:   "",
	}
	err := iptb.TestbedInit(cfg)
	require.NoError(err)

	// Start
	nodes, err := iptb.LoadNodes()
	require.NoError(err)

	err = iptb.TestbedStart(nodes, false, []string{})
	require.NoError(err)

	// Connect
	var wg sync.WaitGroup
	wg.Add(numNodes)
	for i := 0; i < numNodes; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < numNodes; j++ {
				err := iptb.ConnectNodes(nodes[i], nodes[j])
				require.NoError(err)
			}
		}(i)
	}
	wg.Wait()

	// Run
	// everybody mine a block
	for _, n := range nodes {
		out, err := n.RunCmd("go-filecoin", "mining", "once")
		require.NoError(err)

		// TODO make assertions on result
		fmt.Println(out)

		time.Sleep(10 * time.Millisecond)
	}

	// now print yer chains
	for _, n := range nodes {
		out, err := n.RunCmd("go-filecoin", "chain", "ls", "--enc=json")
		require.NoError(err)

		// TODO make assertions on chain state
		fmt.Println(out)
	}

	// kys
	for _, n := range nodes {
		err := n.Kill(true)
		require.NoError(err)
	}
	return

}
