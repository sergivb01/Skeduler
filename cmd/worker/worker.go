package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/client"
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

func (w *worker) run(j jobs.Job) error {
	wsConn, err := streamUploadLogs(context.TODO(), w.host, j.ID)
	if err != nil {
		return fmt.Errorf("getting socket for logs: %w", err)
	}
	defer wsConn.Close()

	logWriter := &websocketWriter{
		wsConn: wsConn,
		mu:     &sync.Mutex{},
		buff:   &bytes.Buffer{},
		host:   w.host,
		id:     j.ID,
	}
	defer logWriter.Close()
	// logging to stderr as well as the custom log io.Writer
	wr := io.MultiWriter(os.Stderr, logWriter)

	t := time.NewTicker(time.Millisecond * 250)
	defer t.Stop()
	go func() {
		for range t.C {
			if err := logWriter.Flush(); err != nil {
				log.Printf("error flushing logs: %v", err)
			}
		}
	}()

	defer func() {
		_, _ = logWriter.Write([]byte(jobs.MagicEnd))
		_, _ = logWriter.Write([]byte{'\n'})
	}()

	log.Printf("[%d] worker running task %+v at %s\n", w.id, t, time.Now())

	return j.Run(context.TODO(), w.cli, w.gpus, wr)
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
