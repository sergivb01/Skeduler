package main

import (
	"context"
	"errors"
	"log"
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
d
func (w *worker) start() {
	for t := range w.reqs {
		log.Printf("[%d] worker running task %+v at %s\n", w.id, t, time.Now())
		if err := t.Run(context.TODO(), w.cli, w.gpus); err != nil {
			log.Printf("error running task: %s", err)
			t.Status = jobs.Cancelled
		} else {
			t.Status = jobs.Finished
		}

		// TODO: update
		if err := updateJob(context.TODO(), t); err != nil {
			log.Printf("failed to update job %+v status: %v\n", t.ID, err)
		}
	}
	w.quit <- struct{}{}
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
