package feed

import (
	"gx/ipfs/QmZH5VXfAJouGMyCCHTRPGCT3e5MG9Lu78Ln3YAYW1XTts/websocket"
)

// Reasons: https://github.com/gorilla/websocket/issues/282#issuecomment-327875136
// "The io.Reader and io.Writer interfaces are standard for byte streams.
// A websocket connection is a stream of messages, not bytes."
type wsWriter struct {
	*websocket.Conn
}

// Writer implements the io.Writer interface on the websocket.
func (w *wsWriter) Write(p []byte) (n int, err error) {
	err = w.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close implements the io.WriteCloser interface on the websocket.
func (w *wsWriter) Close() error {
	return w.Conn.Close()
}
