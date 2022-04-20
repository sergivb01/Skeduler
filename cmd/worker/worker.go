package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log"
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
}

func (w *worker) start() {
	for t := range w.reqs {
		log.Printf("[%d] worker running task %+v at %s\n", w.id, t, time.Now())
		uploader, err := w.logUploader(t.ID.String())
		if err != nil {
			log.Printf("error creating writer: %s", err)
			t.Status = jobs.Cancelled
		} else {
			if err := t.Run(context.TODO(), w.cli, w.gpus, uploader); err != nil {
				log.Printf("error running task: %s", err)
				t.Status = jobs.Cancelled
			} else {
				t.Status = jobs.Finished
			}
		}

		// TODO: update
		if err := updateJob(context.TODO(), t); err != nil {
			log.Printf("failed to update job %+v status: %v\n", t.ID, err)
		}
	}
	w.quit <- struct{}{}
}

type dummyWriter struct {
	id   string
	mu   *sync.RWMutex
	buff *bytes.Buffer
}

func (d *dummyWriter) Write(p []byte) (n int, err error) {

	if d.buff.Len() > 10 {

	}
	return d.buff.Write(p)
}

func (d *dummyWriter) Close() {
}

func (w *worker) logUploader(id string) (io.Writer, error) {
	a := &dummyWriter{
		id:   "",
		mu:   &sync.RWMutex{},
		buff: bytes.NewBuffer(nil),
	}
	logWriter := bufio.NewWriter(a)

	t := time.NewTicker(time.Second)
	defer t.Stop()
	go func() {
		for range t.C {
			_ = logWriter.Flush()
		}
	}()

	defer func() {
		_, _ = logWriter.Write([]byte(jobs.MagicEnd))
		_, _ = logWriter.Write([]byte{'\n'})
		if err := logWriter.Flush(); err != nil {
			log.Printf("error flushing logs for %v: %s\n", id, err)
		}
		// if err := a.Close(); err != nil {
		// 	log.Printf("error closing log file for %v: %s\n", j.ID, err)
		// }
	}()

	return logWriter, nil
}

func puller(tasks chan<- jobs.Job, closing <-chan struct{}) {
	t := time.NewTicker(time.Second)
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
			job, err := fetchJobs(context.TODO())
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
