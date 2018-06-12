package commands

import (
	"testing"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
)

func TestSwarmConnectPeers(t *testing.T) {

	d1 := th.NewDaemon(t, th.SwarmAddr("/ip4/127.0.0.1/tcp/6000")).Start()
	defer d1.ShutdownSuccess()

	d2 := th.NewDaemon(t, th.SwarmAddr("/ip4/127.0.0.1/tcp/6001")).Start()
	defer d2.ShutdownSuccess()

	t.Log("[failure] invalid peer")
	d1.RunFail("invalid peer address",
		"swarm connect /ip4/hello",
	)

	d1.ConnectSuccess(d2)
}
