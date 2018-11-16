package aggregator

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	host "gx/ipfs/QmPMtD39NN63AEUNghk1LFQcTLcCmYL8MtRzdv8BRUsC4Z/go-libp2p-host"
	crypto "gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto"
	net "gx/ipfs/QmQSbtGXCyNrj34LWL8EgXyNNYDZ8r3SwQcpW5pPxVhLnM/go-libp2p-net"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"

	"github.com/filecoin-project/go-filecoin/config"
	"github.com/filecoin-project/go-filecoin/consensus"
	fcmetrics "github.com/filecoin-project/go-filecoin/metrics"
	"github.com/filecoin-project/go-filecoin/testhelpers/iptbtester"
	"github.com/filecoin-project/go-filecoin/tools/aggregator/service/tracker"
	"github.com/filecoin-project/go-filecoin/types"
)

func tipsetGetter() func() consensus.TipSet {
	i := 0
	return func() consensus.TipSet {
		blk := types.NewBlockForTest(nil, uint64(i))
		ts, err := consensus.NewTipSet(blk)
		if err != nil {
			panic(err)
		}
		i++
		return ts
	}
}

type testerService struct {
	S *Service
	T *testing.T

	ConnectCount    int
	DisconnectCount int
}

func mustMakeTesterService(ctx context.Context, t *testing.T) *testerService {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	agg, err := New(ctx, 9081, 0, 0, priv)
	if err != nil {
		t.Fatal(err)
	}

	ts := &testerService{
		S:               agg,
		T:               t,
		ConnectCount:    0,
		DisconnectCount: 0,
	}

	notify := &net.NotifyBundle{
		ListenF:       func(n net.Network, m ma.Multiaddr) {},
		ListenCloseF:  func(n net.Network, m ma.Multiaddr) {},
		ConnectedF:    func(n net.Network, c net.Conn) { ts.ConnectCount++ },
		DisconnectedF: func(n net.Network, c net.Conn) { ts.DisconnectCount++ },
		OpenedStreamF: func(n net.Network, s net.Stream) {},
		ClosedStreamF: func(n net.Network, s net.Stream) {},
	}
	ts.S.Host.Network().Notify(notify)
	return ts
}

func (ts *testerService) MustGetSummary() *tracker.Summary {
	sum, err := ts.S.Tracker.TrackerSummary()
	if err != nil {
		ts.T.Fatal(err)
	}
	return sum
}

type beater struct {
	Host        host.Host
	FullAddress ma.Multiaddr
	Hbs         *fcmetrics.HeartbeatService
	Encoder     *json.Encoder

	test *testing.T
	ctx  context.Context
}

func newBeater(ctx context.Context, t *testing.T, target ma.Multiaddr, hg func() consensus.TipSet) beater {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		panic(err)
	}
	h, err := NewLibp2pHost(context.Background(), priv, 8888)
	if err != nil {
		panic(err)
	}

	fa, err := NewFullAddr(h)
	if err != nil {
		panic(err)
	}
	hbs := fcmetrics.NewHeartbeatService(h, &config.HeartbeatConfig{
		BeatPeriod:      "1.9s",
		ReconnectPeriod: "3s",
		Nickname:        "tester",
		BeatTarget:      target.String(),
	}, hg)
	return beater{
		Host:        h,
		FullAddress: fa,
		Hbs:         hbs,

		test: t,
		ctx:  ctx,
	}
}

func (b *beater) MustConnect() {
	err := b.Hbs.Connect(b.ctx)
	if err != nil {
		b.test.Fatal(err)
	}
	encoder := json.NewEncoder(b.Hbs.Stream())
	b.Encoder = encoder
}

func (b *beater) MustBeat() {
	err := b.Hbs.Beat(b.Encoder)
	if err != nil {
		b.test.Fatal(err)
	}
}

func TestServiceSimple(t *testing.T) {
	//require := require.New(t)
	assert := assert.New(t)

	actx := context.Background()
	bctx := context.Background()
	agg := mustMakeTesterService(actx, t)
	agg.S.Run(actx)
	defer actx.Done()

	tsg1 := tipsetGetter()
	b1 := newBeater(bctx, t, agg.S.FullAddress, tsg1)

	// connect and ensure we get a notification
	b1.MustConnect()
	assert.Equal(1, agg.ConnectCount)
	assert.Equal(0, agg.DisconnectCount)
	assert.Equal(1, len(agg.S.Tracker.TrackedNodes))

	// send some heartbeats
	// TODO must send 2 before tracking is accurate
	b1.MustBeat()
	b1.MustBeat()
	sum := agg.MustGetSummary()
	assert.Equal(1, sum.TrackedNodes)
	assert.Equal(1, sum.NodesInConsensus)
	assert.Equal(0, sum.NodesInDispute)

	// create a new beater and connect in, ensure we get appropriate connect callbacks
	tsg2 := tipsetGetter()
	b2 := newBeater(bctx, t, agg.S.FullAddress, tsg2)
	b2.MustConnect()
	assert.Equal(2, agg.ConnectCount)
	assert.Equal(0, agg.DisconnectCount)
	assert.Equal(2, len(agg.S.Tracker.TrackedNodes))

	// send some heartbeats
	b2.MustBeat()
	b2.MustBeat()
	b1.MustBeat() // keep in-sync with b2

	// generate a tracker summary
	sum = agg.MustGetSummary()
	assert.Equalf(2, sum.TrackedNodes, sum.String())
	assert.Equalf(2, sum.NodesInConsensus, sum.String())
	assert.Equalf(0, sum.NodesInDispute, sum.String())

	// close the connection
	assert.NoError(b1.Hbs.Stream().Conn().Close())
	// Reason for Sleep:
	// https://github.com/libp2p/go-libp2p-swarm/blob/3676e63482ad671539958367c0d14814b6bea542/swarm_conn.go#L42
	time.Sleep(1 * time.Second)
	// we should see a disconnect happen
	assert.Equal(2, agg.ConnectCount)
	assert.Equal(1, agg.DisconnectCount)
	assert.Equal(1, len(agg.S.Tracker.TrackedNodes))

	b2.MustBeat()
	sum = agg.MustGetSummary()
	assert.Equalf(1, sum.TrackedNodes, sum.String())
	assert.Equalf(1, sum.NodesInConsensus, sum.String())
	assert.Equalf(0, sum.NodesInDispute, sum.String())
}

func TestServiceStress(t *testing.T) {
	// Used for manual verification when deving
	t.SkipNow()
	assert := assert.New(t)

	numNodes := 5
	actx := context.Background()
	bctx := context.Background()
	agg := mustMakeTesterService(actx, t)
	agg.S.Run(actx)
	defer actx.Done()

	nodes, err := iptbtester.NewTestNodes(t, numNodes)
	assert.NoError(err)

	defer func() {
		for _, n := range nodes {
			n.Stop(context.Background())
		}
	}()

	for _, n := range nodes {
		// Init each node and connect to the aggregator
		n.MustInit(bctx)
		n.MustStart(bctx)
		n.MustRunCmd(bctx,
			"go-filecoin",
			"config",
			"heartbeat.beatTarget",
			fmt.Sprintf("'%s'", agg.S.FullAddress.String()))

		n.MustRunCmd(bctx,
			"go-filecoin",
			"config",
			"heartbeat.beatPeriod",
			`".1s"`)

		n.MustRunCmd(bctx,
			"go-filecoin",
			"config",
			"heartbeat.reconnectPeriod",
			`".1s"`)
	}
	// wait for connections to occur or timeout
	timeout := time.NewTimer(time.Duration(2*numNodes) * time.Second)
LOOP:
	for {
		select {
		case <-timeout.C:
			t.Fatal("timout waiting for connection")
		default:
			if agg.ConnectCount == numNodes {
				break LOOP
			}
		}
	}
	// the tracker takes a wee bit to become accurate
	time.Sleep(3 * time.Second)
	assert.Equal(numNodes, agg.ConnectCount)
	assert.Equal(0, agg.DisconnectCount)
	sum := agg.MustGetSummary()
	assert.Equalf(numNodes, sum.TrackedNodes, sum.String())
	assert.Equalf(numNodes, sum.NodesInConsensus, sum.String())
	assert.Equalf(0, sum.NodesInDispute, sum.String())

}
