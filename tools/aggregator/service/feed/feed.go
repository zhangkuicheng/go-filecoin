package feed

import (
	"context"
	"net/http"
	"time"

	logging "gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"

	"gx/ipfs/QmZH5VXfAJouGMyCCHTRPGCT3e5MG9Lu78Ln3YAYW1XTts/websocket"

	"github.com/filecoin-project/go-filecoin/tools/aggregator/event"
	"github.com/filecoin-project/go-filecoin/tools/aggregator/service/mw"
)

var log = logging.Logger("feed")

var (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Upgrader for http connections
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Open for anyone to connect to and read
			return true
		},
	}
)

// Feed takes an io.Reader and provides a stream to each connection
type Feed struct {
	// Port the feed listens for connections on
	WSPort int

	ctx     context.Context
	src     chan event.HeartbeatEvent // TODO make this type the same as EvtChan in service
	mirrorw *mw.MirrorWriter
}

// NewFeed creates a new Feed, Feed's Run method will read from chan `source`, see
// Run for details.
func NewFeed(ctx context.Context, sp int, source chan event.HeartbeatEvent) *Feed {
	return &Feed{
		ctx:     ctx,
		src:     source,
		mirrorw: mw.NewMirrorWriter(),
		WSPort:  sp,
	}
}

// ServeHTTP upgrades a connection to a websocket and writes a stream from
// the feed to each
func (f *Feed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("new http connection %s", r.RemoteAddr)

	// upgrade http connection to a websocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("failed to upgrade connection: %s", err)
		return
	}

	// wrap the websocket with wsWriter which implements the writer interface.
	wrapper := &wsWriter{conn}
	// the mirror writer writes to all attached writers.
	f.mirrorw.AddWriter(wrapper)

	// ping messages are used to verify that the client is still connected.
	go func() {
		// This ticker is used to send ping messages to the client on a regular basis
		ticker := time.NewTicker(pingPeriod)

		defer func() {
			log.Infof("closing http connection %s", conn.RemoteAddr())
			ticker.Stop()
			conn.Close() // nolint: errcheck
		}()

		conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint: errcheck
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})

		for range ticker.C {
			log.Info("pinging websocket")
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				// This means we have lost our connection to the client.
				// The mirror writer will detect that the connection is no
				// longer writable and prune it so we can just return.
				log.Errorf("failed ping to websocket: %s", err)
				return
			}
		}
	}()
}

// Run writes messages from `feed.src` to feed's mirror writer, which will
// broadcast the written bytes to all Writers, see MirrorWritter for details.
func (f *Feed) Run() {
	for {
		select {
		// A heartbeat is sent
		case hb := <-f.src:
			hbb := hb.MustMarshalJSON()
			_, err := f.mirrorw.Write(hbb)
			if err != nil {
				log.Errorf("failed to broadcast heartbeat to writers: %s", err)
			}
		case <-f.ctx.Done():
			log.Infof("Run context done")
			f.mirrorw.Close() // nolint: errcheck
			return
		}

	}
}
