package main

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/client"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type worker struct {
	id   int
	cli  *client.Client
	db   database.Database
	reqs <-chan jobs.Job
	quit chan<- struct{}
	gpus []string
}

func (w *worker) start() {
	for t := range w.reqs {
		log.Printf("[%d] worker running task %+v at %s\n", w.id, t, time.Now())
		if err := t.Run(context.TODO(), w.cli, w.gpus); err != nil {
			log.Printf("error running task: %s", err)
			t.Status = jobs.Cancelled
		} else {
			t.Status = jobs.Finished
		}

		if err := w.db.Update(context.TODO(), &t); err != nil {
			log.Printf("failed to update job %+v status: %v\n", t.ID, err)
		}
	}
	w.quit <- struct{}{}
}

func puller(tasks chan<- jobs.Job, closing <-chan struct{}, db database.Database) {
	t := time.Tick(time.Second * 3)
	for {
		select {
		case <-closing:
			return
		case <-t:
			// all workers are being used
			if len(tasks) == cap(tasks) {
				continue
			}

			job, err := db.FetchJob(context.TODO())
			if err != nil {
				log.Printf("error pulling: %v\n", err)
				continue
			}

			// no job available
			if job == nil {
				continue
			}
			tasks <- *job

			break
		}
	}
}
