package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type worker struct {
	id   int
	cli  *client.Client
	reqs <-chan jobs.Job
	quit chan<- struct{}
	gpus []string
	host string
}

func (w *worker) start() {
	for t := range w.reqs {
		if err := w.run(t); err != nil {
			log.Printf("error running task: %s", err)
			t.Status = jobs.Cancelled
		} else {
			t.Status = jobs.Finished
		}

		if err := updateJob(context.TODO(), w.host, t); err != nil {
			log.Printf("failed to update job %+v status: %v\n", t.ID, err)
		}
	}
	w.quit <- struct{}{}
}

type dummyWriter struct {
	wsConn *websocket.Conn
	mu     *sync.Mutex
	buff   *bytes.Buffer
}

var _ io.Writer = &dummyWriter{}

func (d *dummyWriter) Write(b []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.buff.Write(b)
}

func (d *dummyWriter) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.buff.Len() == 0 {
		return nil
	}

	b := d.buff.Bytes()
	log.Printf("sending %d bytes to websocket!", len(b))
	if err := d.wsConn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return err
	}
	d.buff.Reset()

	return nil
}

func (w *worker) run(j jobs.Job) error {
	wsConn, err := streamUploadLogs(context.TODO(), w.host, j.ID)
	if err != nil {
		return fmt.Errorf("getting socket for logs: %w", err)
	}
	defer wsConn.Close()

	logWriter := &dummyWriter{
		wsConn: wsConn,
		mu:     &sync.Mutex{},
		buff:   &bytes.Buffer{},
	}

	t := time.NewTicker(time.Millisecond * 250)
	defer t.Stop()
	go func() {
		for range t.C {
			_ = logWriter.Flush()
		}
		log.Printf("finished log writing at %s", time.Now())
	}()

	defer func() {
		_, _ = logWriter.Write([]byte(jobs.MagicEnd))
		_, _ = logWriter.Write([]byte{'\n'})
		if err := logWriter.Flush(); err != nil {
			log.Printf("error flushing logs for %v: %s\n", j.ID, err)
		}
		log.Printf("finished defer at %s", time.Now())
	}()

	log.Printf("[%d] worker running task %+v at %s\n", w.id, t, time.Now())

	return j.Run(context.TODO(), w.cli, w.gpus, logWriter)
}

func puller(tasks chan<- jobs.Job, closing <-chan struct{}, host string) {
	t := time.NewTicker(time.Second * 3)
	defer t.Stop()

	for {
		select {
		case <-closing:
			return
		case <-t.C:
			// all workers are being used
			if len(tasks) == cap(tasks) {
				continue
			}

			// TODO: fix
			job, err := fetchJobs(context.TODO(), host)
			if err != nil {
				// no job available
				if errors.Is(err, errNoJob) {
					continue
				}
				log.Printf("error pulling: %v\n", err)
				continue
			}

			tasks <- job

			break
		}
	}
}
