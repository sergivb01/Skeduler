package main

import (
	"bytes"
	"io"
	"sync"

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

func (d *websocketWriter) Write(b []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.buff.Write(b)
}

// Close directly calls the Flush method
func (d *websocketWriter) Close() error {
	return d.Flush()
}

func (d *websocketWriter) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.buff.Len() == 0 {
		return nil
	}

	b := d.buff.Bytes()
	if err := d.wsConn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return err
	}
	d.buff.Reset()

	return nil
}
