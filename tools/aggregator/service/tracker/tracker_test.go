package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectNode(t *testing.T) {
	assert := assert.New(t)
	tracker := NewTracker(0)

	peer1 := "peer1"
	peer2 := "peer2"

	tracker.ConnectNode(peer1)
	assert.Equal(1, len(tracker.TrackedNodes))

	tracker.ConnectNode(peer2)
	assert.Equal(2, len(tracker.TrackedNodes))

	tracker.DisconnectNode(peer1)
	assert.Equal(1, len(tracker.TrackedNodes))

	tracker.DisconnectNode(peer2)
	assert.Equal(0, len(tracker.TrackedNodes))
}

func TestConsensus(t *testing.T) {
	assert := assert.New(t)
	tracker := NewTracker(0)

	peer1 := "peer1"
	peer2 := "peer2"
	ts1 := "tipset1"
	ts2 := "tipset2"

	tracker.ConnectNode(peer1)
	tracker.TrackConsensus(peer1, ts1)
	assert.Equal(1, tracker.TipsCount[ts1])
	assert.Equal(1, len(tracker.TrackedNodes))
	assert.Equal(ts1, tracker.NodeTips[peer1])

	sum, err := tracker.TrackerSummary()
	assert.NoError(err)
	assert.Equal(&Summary{
		TrackedNodes:     1,
		NodesInConsensus: 1,
		NodesInDispute:   0,
		HeaviestTipset:   ts1,
	},
		sum,
	)

	tracker.ConnectNode(peer2)
	tracker.TrackConsensus(peer2, ts1)
	assert.Equal(2, tracker.TipsCount[ts1])
	assert.Equal(2, len(tracker.TrackedNodes))
	assert.Equal(ts1, tracker.NodeTips[peer1])
	assert.Equal(ts1, tracker.NodeTips[peer2])

	sum, err = tracker.TrackerSummary()
	assert.NoError(err)
	assert.Equal(&Summary{
		TrackedNodes:     2,
		NodesInConsensus: 2,
		NodesInDispute:   0,
		HeaviestTipset:   ts1,
	},
		sum,
	)

	tracker.TrackConsensus(peer1, ts2)
	assert.Equal(1, tracker.TipsCount[ts1])
	assert.Equal(1, tracker.TipsCount[ts2])
	assert.Equal(2, len(tracker.TrackedNodes))
	assert.Equal(ts2, tracker.NodeTips[peer1])
	assert.Equal(ts1, tracker.NodeTips[peer2])

	sum, err = tracker.TrackerSummary()
	assert.NoError(err)
	assert.Equal(&Summary{
		TrackedNodes:     2,
		NodesInConsensus: 0,
		NodesInDispute:   2,
		HeaviestTipset:   "",
	},
		sum,
	)

	tracker.TrackConsensus(peer2, ts2)
	assert.Equal(0, tracker.TipsCount[ts1])
	assert.Equal(2, tracker.TipsCount[ts2])
	assert.Equal(2, len(tracker.TrackedNodes))
	assert.Equal(ts2, tracker.NodeTips[peer1])
	assert.Equal(ts2, tracker.NodeTips[peer2])

	sum, err = tracker.TrackerSummary()
	assert.NoError(err)
	assert.Equal(&Summary{
		TrackedNodes:     2,
		NodesInConsensus: 2,
		NodesInDispute:   0,
		HeaviestTipset:   ts2,
	},
		sum,
	)

}
