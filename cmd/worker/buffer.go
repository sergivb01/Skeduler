package main

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

// websocketWriter is an implementation io.Writer that needs the developer to call
// the Flush() method periodically so that the underlying buffer contents are sent
// to the websocket connection. Messages are sent as websocket.BinaryMessage
type websocketWriter struct {
	wsConn *websocket.Conn
	mu     *sync.Mutex
	buff   *bytes.Buffer
}

var _ io.WriteCloser = &websocketWriter{}

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
			// TODO: handle reconnect
			return nil
		}
		return err
	}
	w.buff.Reset()

	return nil
}
