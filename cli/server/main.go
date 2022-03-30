package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/client"
)

var wg = &sync.WaitGroup{}

func main() {
	conf, err := FromFile("config.yml")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", conf)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	tasks := make(chan JobRequest)
	done := make(chan struct{}, len(conf.Queues))
	for i, q := range conf.Queues {
		go startWorker(cli, tasks, done, i, q.GPUs)
	}

	httpClose := make(chan struct{}, 1)
	go func() {
		err := startHttp(tasks, httpClose, conf.Http)
		if err != nil {
			log.Printf("error server: %s\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c

	log.Printf("starting gracefull shutdown. Waiting for all pending tasks to finish\n")

	// close http server
	httpClose <- struct{}{}

	// wait for running experiments to finish
	wg.Wait()

	// close workers, they are all done now
	close(tasks)

	// wait for workers to close
	for range conf.Queues {
		<-done
	}

	log.Printf("shutdown!\n")
}

func schedule(tasks chan<- JobRequest, j JobRequest) {
	go func() {
		wg.Add(1)
		tasks <- j
		wg.Done()
	}()
}

func startWorker(cli *client.Client, reqs <-chan JobRequest, quit chan<- struct{}, id int, gpus []string) {
	for t := range reqs {
		log.Printf("[%d] worker running task %+v at %s\n", id, t, time.Now())
		if err := t.Run(context.TODO(), cli, gpus); err != nil {
			log.Printf("error running task: %s", err)
		}
	}
	quit <- struct{}{}
}
