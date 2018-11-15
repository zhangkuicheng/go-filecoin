package feed

import (
	"context"
	"fmt"
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

// Feed reads HeartbearEvents from a channel and writes them to all writers in
// its mirror writer. A writer in the mirror writer represensts a websocket connection.
type Feed struct {
	// Port the feed listens for connections on
	WSPort int

	ctx     context.Context
	src     event.Evtch
	mirrorw *mw.MirrorWriter
}

// NewFeed creates a new Feed, Feed's run method will read from chan `source`, see
// run for details.
func NewFeed(ctx context.Context, sp int, source event.Evtch) *Feed {
	return &Feed{
		ctx:     ctx,
		src:     source,
		mirrorw: mw.NewMirrorWriter(),
		WSPort:  sp,
	}
}

// StartHandler sets-up the Feeds http handlers and runs the feed.
func (f *Feed) StartHandler() {
	http.Handle("/feed", f)
	go f.run()
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", f.WSPort), nil); err != nil {
			log.Fatal(err)
		}
	}()
	log.Debug("setup feed handlers")
}

// ServeHTTP upgrades a connection to a websocket and writes a stream from
// the feed to each.
func (f *Feed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("new http connection %s", r.RemoteAddr)

	// upgrade http connection to a websocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warningf("failed to upgrade connection: %s", err)
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

		conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint: errcheck
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})

		defer func() {
			log.Infof("closing http connection %s", r.RemoteAddr)
			ticker.Stop()
			wrapper.Close() // nolint: errcheck
		}()

		for {
			select {
			case <-f.ctx.Done():
				return
			case <-ticker.C:
				log.Debugf("pinging websocket: %v", conn.RemoteAddr().String())
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
					// This means we have lost our connection to the client.
					// The mirror writer will detect that the connection is no
					// longer writable and prune it so we can just return.
					log.Debugf("failed ping to websocket: %s", err)
					return
				}

			}
		}
	}()
}

// run writes messages from `feed.src` to feed's mirror writer, which will
// broadcast the written bytes to all Writers, see MirrorWritter for details.
func (f *Feed) run() {
	for {
		select {
		// A heartbeat is sent
		case hb := <-f.src:
			hbb := hb.MustMarshalJSON()
			_, err := f.mirrorw.Write(hbb)
			if err != nil {
				log.Warningf("failed to broadcast heartbeat to writers: %s", err)
			}
		case <-f.ctx.Done():
			log.Infof("run context done")
			f.mirrorw.Close() // nolint: errcheck
			return
		}

	}
}
