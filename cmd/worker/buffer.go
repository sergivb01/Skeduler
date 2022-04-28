package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

// websocketWriter is an implementation io.Writer that needs the developer to call
// the Flush() method periodically so that the underlying buffer contents are sent
// to the websocket connection. Messages are sent as websocket.BinaryMessage
type websocketWriter struct {
	uri string

	mu     *sync.Mutex
	wsConn *websocket.Conn
	buff   *bytes.Buffer
}

var errConnectionLost = errors.New("connection to websocket server is down")

var _ io.WriteCloser = &websocketWriter{}

func (w *websocketWriter) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(w.uri, nil)
	if err != nil {
		return fmt.Errorf("dialing websocket: %w", err)
	}
	w.wsConn = conn
	return nil
}

func (w *websocketWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buff.Write(b)
}

// Close directly calls the Flush method
func (w *websocketWriter) Close() error {
	return w.Flush()
}

func (w *websocketWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buff.Len() == 0 {
		return nil
	}

	b := w.buff.Bytes()
	if err := w.wsConn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		if errors.Is(err, syscall.EPIPE) {
			if err := w.connect(); err != nil {
				return errConnectionLost
			}

			// we have reconnected to websocket, but will need to do this process again
			return nil
		}
		return err
	}
	w.buff.Reset()

	return nil
}
