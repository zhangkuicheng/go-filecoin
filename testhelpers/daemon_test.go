package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	sm "github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestDaemon struct {
	*Daemon

	test *testing.T
}

func NewTestDaemon(t *testing.T, options ...func(*Daemon)) *TestDaemon {
	daemon, err := NewDaemon()
	assert.NoError(t, err)
	return &TestDaemon{daemon, t}
}

func (td *TestDaemon) Start() *TestDaemon {
	_, err := td.Daemon.Start()
	assert.NoError(td.test, err)
	return td
}

func (td *TestDaemon) ShutdownSuccess() {
	assert.NoError(td.test, td.Daemon.Shutdown())
}

func TestDaemonStartupMessage(t *testing.T) {
	assert := assert.New(t)
	daemon := NewTestDaemon(t).Start()
	daemon.ShutdownSuccess()

	out := daemon.ReadStdout()
	assert.Regexp("^My peer ID is [a-zA-Z0-9]*", out)
	assert.Regexp("\\nSwarm listening on.*", out)
}

func TestSwarmConnectPeers(t *testing.T) {

	d1 := NewTestDaemon(t, SwarmAddr("/ip4/127.0.0.1/tcp/6000")).Start()
	defer d1.ShutdownSuccess()

	d2 := NewTestDaemon(t, SwarmAddr("/ip4/127.0.0.1/tcp/6001")).Start()
	defer d2.ShutdownSuccess()

	_, err := d1.Connect(d2.Daemon)
	assert.NoError(t, err)

	d3 := NewTestDaemon(t).Start()
	defer d3.ShutdownSuccess()

	d4 := NewTestDaemon(t).Start()
	defer d4.ShutdownSuccess()

	_, err = d1.Connect(d3.Daemon)
	assert.NoError(t, err)

	_, err = d1.Connect(d4.Daemon)
	assert.NoError(t, err)

	_, err = d2.Connect(d3.Daemon)
	assert.NoError(t, err)

	_, err = d2.Connect(d4.Daemon)
	assert.NoError(t, err)

	_, err = d3.Connect(d4.Daemon)
	assert.NoError(t, err)
}

func TestMinerCreateAddr(t *testing.T) {
	require := require.New(t)

	d1 := NewTestDaemon(t, SwarmAddr("/ip4/127.0.0.1/tcp/6000")).Start()
	defer func() {
		d1.ShutdownSuccess()
		t.Log(d1.ReadStderr())
		t.Log(d1.ReadStdout())
	}()

	d2 := NewTestDaemon(t, SwarmAddr("/ip4/127.0.0.1/tcp/6001")).Start()
	defer func() {
		d2.ShutdownSuccess()
		t.Log(d2.ReadStderr())
		t.Log(d2.ReadStdout())
	}()

	_, err := d1.Connect(d2.Daemon)
	require.NoError(err)

	w1Addr, err := d1.CreateWalletAddr()
	require.NoError(err)
	require.NotEmpty(w1Addr)

	w2Addr, err := d2.CreateWalletAddr()
	require.NoError(err)
	require.NotEmpty(w2Addr)

	require.NoError(d1.MiningOnce())
	require.NoError(d2.MiningOnce())

	m1Addr, err := d1.CreateMinerAddr()
	if err != nil {
		d1.Shutdown()
		t.Log(d1.ReadStderr())
		t.Log(d1.ReadStdout())
	}
	require.NoError(err)
	require.NotEmpty(m1Addr)

	m2Addr, err := d2.CreateMinerAddr()
	require.NoError(err)
	require.NotEmpty(m2Addr)

}

func TestOrderbookListing(t *testing.T) {
	require := require.New(t)
	client := NewTestDaemon(t, SwarmAddr("/ip4/127.0.0.1/tcp/6000")).Start()
	defer func() {
		client.ShutdownSuccess()
		//t.Log(client.ReadStderr())
		//t.Log(client.ReadStdout())
	}()

	miner := NewTestDaemon(t, SwarmAddr("/ip4/127.0.0.1/tcp/6001")).Start()
	defer func() {
		miner.ShutdownSuccess()
		//t.Log(miner.ReadStderr())
		//t.Log(miner.ReadStdout())
	}()

	_, err := client.Connect(miner.Daemon)
	require.NoError(err)

	/*
			w1Addr, err := client.CreateWalletAddr()
			require.NoError(err)
			require.NotEmpty(w1Addr)

		w2Addr, err := miner.CreateWalletAddr()
		require.NoError(err)
		require.NotEmpty(w2Addr)
	*/

	require.NoError(client.MiningOnce())
	require.NoError(miner.MiningOnce())

	/*
		m1Addr, err := client.CreateMinerAddr()
		if err != nil {
			client.Shutdown()
			t.Log(client.ReadStderr())
			t.Log(client.ReadStdout())
		}
		require.NoError(err)
		require.NotEmpty(m1Addr)
	*/

	m2Addr, err := miner.CreateMinerAddr()
	require.NoError(err)
	require.NotEmpty(m2Addr)

	// ensure they have an addr they can bid from
	clientfrom, err := client.GetMainWalletAddress()
	require.NoError(err)

	miner.Run("mining", "start")
	err = client.ClientAddBid(context.TODO(), clientfrom, 10, 10)
	require.NoError(err)
	miner.Run("mining", "stop")

	miner.Run("mining", "start")
	err = miner.MinerAddAsk(context.TODO(), m2Addr.String(), 10, 10)
	require.NoError(err)
	miner.Run("mining", "stop")

	out, err := client.OrderbookGetAsks(context.TODO())
	require.NoError(err)
	asks := extractAsks(out.ReadStdout())

	out, err = client.OrderbookGetBids(context.TODO())
	require.NoError(err)
	bids := extractUnusedBids(out.ReadStdout())

	fmt.Println(asks[0].ID)
	fmt.Println(bids[0].ID)
	psudoData := "QmTz3oc4gdpRMKP2sdGUPZTAGRngqjsi99BPoztyP53JMM"

	out, err = client.ProposeDeal(asks[0].ID, bids[0].ID, psudoData)
	require.NoError(err)
	fmt.Println(out.ReadStdout())
	miner.MiningOnce()
	client.MiningOnce()

	out, err = client.OrderbookGetDeals(context.TODO())
	require.NoError(err)
	time.Sleep(3 * time.Minute)
	deals := extractDeals(out.ReadStdout())
	fmt.Println(deals)

}

func extractAsks(input string) []sm.Ask {

	// remove last new line
	o := strings.Trim(input, "\n")
	// separate ndjson on new lines
	as := strings.Split(o, "\n")
	fmt.Println(as)
	fmt.Println(len(as))

	var asks []sm.Ask
	for _, a := range as {
		var ask sm.Ask
		fmt.Println(a)
		err := json.Unmarshal([]byte(a), &ask)
		if err != nil {
			panic(err)
		}
		asks = append(asks, ask)
	}
	return asks
}

func extractUnusedBids(input string) []sm.Bid {
	// remove last new line
	o := strings.Trim(input, "\n")
	// separate ndjson on new lines
	bs := strings.Split(o, "\n")
	fmt.Println(bs)
	fmt.Println(len(bs))

	var bids []sm.Bid
	for _, b := range bs {
		var bid sm.Bid
		fmt.Println(b)
		err := json.Unmarshal([]byte(b), &bid)
		if err != nil {
			panic(err)
		}
		if bid.Used {
			continue
		}
		bids = append(bids, bid)
	}
	return bids
}

func extractDeals(input string) []sm.Deal {

	// remove last new line
	o := strings.Trim(input, "\n")
	// separate ndjson on new lines
	ds := strings.Split(o, "\n")
	fmt.Println(ds)
	fmt.Println(len(ds))

	var deals []sm.Deal
	for _, d := range ds {
		var deal sm.Deal
		fmt.Println(d)
		err := json.Unmarshal([]byte(d), &deal)
		if err != nil {
			panic(err)
		}
		deals = append(deals, deal)
	}
	return deals
}

// Gor debugging
/*
func TestDaemonEventLogs(t *testing.T) {
	assert := assert.New(t)
	daemon := NewTestDaemon(t).Start()
	defer daemon.ShutdownSuccess()

	t.Log("setting up event log stream")
	logs := daemon.EventLogStream()
	blocks := 10

	done := make(chan struct{}, 2)
	go func() {
		d := json.NewDecoder(logs)
		var m map[string]interface{}

		blocksLeft := blocks
		eventsSeen := 0
		for ; blocksLeft > 0; eventsSeen++ {
			err := d.Decode(&m)
			assert.NoError(err)

			if m["Operation"].(string) == "AddNewBlock" {
				blocksLeft--
			}
		}

		t.Logf("Parsed %d events", eventsSeen)
		done <- struct{}{}
	}()

	go func() {
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < blocks; i++ {
			t.Log("mining once...")
			err := daemon.MiningOnce()
			assert.NoError(err)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		select {
		case <-done:
			return // success
		case <-time.After(5 * time.Second):
			t.Fail()
		}
	case <-time.After(5 * time.Second):
		t.Fail()
	}
}
*/
