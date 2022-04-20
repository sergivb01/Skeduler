package main

import (
	"context"
	"time"

	"github.com/tj/go-buffer"
)

type logWriter struct {
}

func test() {
	b := buffer.New(buffer.WithFlushHandler(func(ctx context.Context, elems []interface{}) error {
		return nil
	}), buffer.WithFlushInterval(time.Second))
	b.Flush()
}
