package event

import (
	"encoding/json"
	"time"

	peer "gx/ipfs/QmQsErDt8Qgw1XrsXf2BpEzDgGWtB1YLsTAARBup5b6B9W/go-libp2p-peer"

	fcmetrics "github.com/filecoin-project/go-filecoin/metrics"
)

// HeartbeatEvent contains a heartbeat, the time it was received and who it was from
type HeartbeatEvent struct {
	// FromPeer is who created the event
	FromPeer peer.ID `json:"fromPeer"`
	// ReceivedTimestamp represents when the event was received
	ReceivedTimestamp time.Time `json:"receivedTimestamp"`
	// Heartbeat data sent by `FromPeer`
	Heartbeat fcmetrics.Heartbeat `json:"heartbeat"`
}

// MustMarshalJSON marshals a HeartbeatEvent to json, and panics if marshaling fails.
func (t HeartbeatEvent) MustMarshalJSON() []byte {
	event := t.getJSONMap()
	out, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	return out
}

func (t HeartbeatEvent) getJSONMap() map[string]interface{} {
	event := map[string]interface{}{
		"receivedTimestamp": t.ReceivedTimestamp.UTC(),
		"fromPeer":          t.FromPeer,
		"heartbeat":         t.Heartbeat,
	}
	return event
}
