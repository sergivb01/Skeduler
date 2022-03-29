package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/docker/docker/client"
)

const N_WORKERS = 2

func runExample(tasks chan<- JobRequest) {
	t, err := NewFromFile("example.json")
	if err != nil {
		panic(err)
	}
	log.Printf("waiting for 3s\n")
	time.Sleep(time.Second * 3)

	log.Printf("scheduling...\n")
	tasks <- *t
	tasks <- *t
	tasks <- *t

	time.Sleep(time.Second)
	tasks <- *t
	tasks <- *t
	tasks <- *t
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	tasks := make(chan JobRequest)

	go runExample(tasks)

	quit := make(chan struct{}, N_WORKERS)
	for i := 0; i < N_WORKERS; i++ {
		go startWorker(cli, tasks, quit, "all")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	for i := 0; i < N_WORKERS; i++ {
		quit <- struct{}{}
	}

	fmt.Printf("shutdown!\n")
}

func startWorker(cli *client.Client, reqs <-chan JobRequest, quit <-chan struct{}, gpu string) {
	for {
		select {
		case t := <-reqs:
			log.Printf("[%s] worker running task\n", time.Now())
			if err := t.Run(context.TODO(), cli, []string{gpu}); err != nil {
				log.Printf("error running task: %s", err)
			}
			break
		case <-quit:
			return
		}
	}
}
