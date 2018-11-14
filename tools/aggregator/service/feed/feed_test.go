package feed

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metrics "github.com/filecoin-project/go-filecoin/metrics"
	th "github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/filecoin-project/go-filecoin/tools/aggregator/event"
)

func TestFeed(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	pr, pw := io.Pipe()
	ctx := context.Background()
	sourceCh := make(chan event.HeartbeatEvent)

	// make a feed
	feed := NewFeed(ctx, 0, sourceCh)
	// start the feed
	go feed.Run()
	// add a writer that events will be written to
	feed.mirrorw.AddWriter(pw)

	hb := event.HeartbeatEvent{
		FromPeer:          th.RequireRandomPeerID(),
		ReceivedTimestamp: time.Now().UTC(),
		Heartbeat: metrics.Heartbeat{
			Head:     "head",
			Height:   0,
			Nickname: "poo",
		},
	}
	// send an event to the feed
	sourceCh <- hb

	// decode the event
	decoder := json.NewDecoder(pr)
	var hbe event.HeartbeatEvent
	require.NoError(decoder.Decode(&hbe))

	// heartbeats should match
	assert.Equal(hb.Heartbeat, hbe.Heartbeat)

}

func TestFeedMultiConns(t *testing.T) {
	require := require.New(t)

	pr0, pw0 := io.Pipe()
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	pr4, pw4 := io.Pipe()
	ctx := context.Background()
	sourceCh := make(chan event.HeartbeatEvent)

	// make a feed
	feed := NewFeed(ctx, 0, sourceCh)
	// start the feed
	go feed.Run()
	// add writers that events will be written to
	feed.mirrorw.AddWriter(pw0)
	feed.mirrorw.AddWriter(pw1)
	feed.mirrorw.AddWriter(pw2)
	feed.mirrorw.AddWriter(pw3)
	feed.mirrorw.AddWriter(pw4)

	for n := 0; n < 10000; n++ {
		hb := event.HeartbeatEvent{
			FromPeer:          th.RequireRandomPeerID(),
			ReceivedTimestamp: time.Now().UTC(),
			Heartbeat: metrics.Heartbeat{
				Head:     "head",
				Height:   uint64(n),
				Nickname: "poo",
			},
		}
		// send an event to the feed
		sourceCh <- hb

		// decode the event from all readers
		var hbe event.HeartbeatEvent
		decoder0 := json.NewDecoder(pr0)
		require.NoError(decoder0.Decode(&hbe))

		decoder1 := json.NewDecoder(pr1)
		require.NoError(decoder1.Decode(&hbe))

		decoder2 := json.NewDecoder(pr2)
		require.NoError(decoder2.Decode(&hbe))

		decoder3 := json.NewDecoder(pr3)
		require.NoError(decoder3.Decode(&hbe))

		decoder4 := json.NewDecoder(pr4)
		require.NoError(decoder4.Decode(&hbe))
	}
}
