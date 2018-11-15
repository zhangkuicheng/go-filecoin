package tracker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	logging "gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var log = logging.Logger("aggregator/tracker")

const aggregatorLabel = "aggregator"

var (
	connectedNodes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "connected_nodes",
			Help: "number of nodes connected to aggregator",
		},
		[]string{aggregatorLabel},
	)

	nodesConsensus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nodes_in_consensus",
			Help: "number of nodes in consensus",
		},
		[]string{aggregatorLabel},
	)

	nodesDispute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nodes_in_dispute",
			Help: "number of nodes in dispute",
		},
		[]string{aggregatorLabel},
	)
)

// Tracker tracks node consensus from heartbeats
type Tracker struct {
	// NodeTips is a mapping from peerID's to Tipsets
	NodeTips map[string]string
	// TipsCount is a mapping from tipsets to number of peers mining on said tipset.
	TipsCount map[string]int
	// TrackedNodes is the set of nodes currently connected to the aggregator, this
	// value is updated using the net.NotifyBundle in service_utils.go
	TrackedNodes map[string]struct{}

	// mutex that protects access to the fields in Tracker:
	// - NodeTips
	// - TipsCount
	// - TrackedNodes
	mux sync.Mutex

	metricsP int
}

// Summary represents the information a tracker has on the nodes
// its receiving heartbeats from
type Summary struct {
	TrackedNodes     int
	NodesInConsensus int
	NodesInDispute   int
	HeaviestTipset   string
}

func (s *Summary) String() string {
	return fmt.Sprintf("NumNodes: %d, Consensus: %d, Dispute: %d, Head: %s",
		s.TrackedNodes, s.NodesInConsensus, s.NodesInDispute, s.HeaviestTipset)
}

// NewTracker initializes a tracker
func NewTracker(mp int) *Tracker {
	return &Tracker{
		TrackedNodes: make(map[string]struct{}),
		NodeTips:     make(map[string]string),
		TipsCount:    make(map[string]int),
		metricsP:     mp,
	}
}

func (t *Tracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var peers []string
	for p := range t.TrackedNodes {
		peers = append(peers, p)
	}
	js, err := json.Marshal(peers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js) // nolint: errcheck
}

// StartHandler sets-up the Trackers http handlers.
func (t *Tracker) StartHandler() {
	// register the prometheus metrics and configure an endpoint to query them.
	prometheus.MustRegister(connectedNodes, nodesConsensus, nodesDispute)
	http.Handle("/metrics", promhttp.Handler())

	// tracer needs to report number of connected nodes, register it, see ServeHTTP
	// impl for details on reporting.
	http.Handle("/report", t)

	// now serve the aformentioend endpoints
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", t.metricsP), nil); err != nil {
			log.Fatal(err)
		}
	}()

	// the tracker needs to update the prometheus metrics it serves.
	updateMetrics := time.NewTicker(time.Second * 2)
	go func() {
		for range updateMetrics.C {
			sum, err := t.TrackerSummary()
			if err != nil {
				log.Info("tracker not ready, waiting to receieve tipsets")
				continue
			}
			log.Infof("tracker status: %s", sum.String())
		}
	}()
	log.Debug("setup tracker handlers")
}

// ConnectNode will add a node to the trackers `TrackedNode` set and
// increment the connected_nodes prometheus metric.
func (t *Tracker) ConnectNode(peer string) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.TrackedNodes[peer] = struct{}{}
	connectedNodes.WithLabelValues(aggregatorLabel).Inc()
}

// DisconnectNode will remove a node from the trackers `TrackedNode` set and
// decrement the connected_nodes prometheus metric.
func (t *Tracker) DisconnectNode(peer string) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if _, ok := t.TrackedNodes[peer]; !ok {
		log.Warningf("received disconnect from unknown peer: %s", peer)
		return
	}

	delete(t.TrackedNodes, peer)

	curTs, ok := t.NodeTips[peer]
	if ok {
		t.TipsCount[curTs]--

		if t.TipsCount[curTs] == 0 {
			delete(t.TipsCount, curTs)
		}

	}

	delete(t.NodeTips, peer)

	connectedNodes.WithLabelValues(aggregatorLabel).Dec()
}

// TrackConsensus updates the metrics Tracker keeps, threadsafe
func (t *Tracker) TrackConsensus(peer, ts string) {
	log.Debugf("track peer: %s, tipset: %s", peer, ts)
	t.mux.Lock()
	defer t.mux.Unlock()

	if _, ok := t.TrackedNodes[peer]; !ok {
		// TODO this is a hack because I think libp2p notifee is broken
		log.Warningf("Received heartbeat from unknown peer: %s", peer)
		log.Infof("adding peer to TrackedNodes list: %v", t.TrackedNodes)
		t.TrackedNodes[peer] = struct{}{}

		connectedNodes.WithLabelValues(aggregatorLabel).Set(float64(len(t.TrackedNodes)))
	}

	// get the tipset the nodes is currently on.
	curTs, ok := t.NodeTips[peer]
	if ok {
		t.TipsCount[curTs]--
		if t.TipsCount[curTs] == 0 {
			delete(t.TipsCount, curTs)
		}
	}

	t.NodeTips[peer] = ts
	t.TipsCount[ts]++
}

// TrackerSummary generates a summary of the metrics Tracker keeps, threadsafe
func (t *Tracker) TrackerSummary() (*Summary, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	tn := len(t.TrackedNodes)
	nc, ht, err := nodesInConsensus(t.TipsCount)
	if err != nil {
		return nil, err
	}
	nd := tn - nc

	nodesConsensus.WithLabelValues(aggregatorLabel).Set(float64(nc))
	nodesDispute.WithLabelValues(aggregatorLabel).Set(float64(nd))
	return &Summary{
		TrackedNodes:     tn,
		NodesInConsensus: nc,
		NodesInDispute:   nd,
		HeaviestTipset:   ht,
	}, nil
}
