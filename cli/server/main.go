package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
)

const N_WORKERS = 2

func runExample(jobs chan<- JobRequest) {
	t, err := NewFromFile("example.json")
	if err != nil {
		panic(err)
	}
	log.Printf("waiting for 1s\n")
	time.Sleep(time.Second * 1)

	log.Printf("scheduling...\n")
	for i := 0; i < 5; i++ {
		schedule(jobs, *t)
	}
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	tasks := make(chan JobRequest)
	done := make(chan struct{}, N_WORKERS)
	for i := 0; i < N_WORKERS; i++ {
		go startWorker(cli, tasks, done, "all")
	}

	go runExample(tasks)
	httpClose := make(chan struct{}, 1)
	go func() {
		err := startHttp(tasks, httpClose)
		if err != nil {
			log.Printf("error server: %s\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c
	httpClose <- struct{}{}
	log.Printf("Starting gracefull shutdown. Waiting for all pending tasks to finish\n")

	wg.Wait()
	// close and wait for workers to finish current running jobs
	close(tasks)
	for i := 0; i < N_WORKERS; i++ {
		<-done
	}

	fmt.Printf("shutdown!\n")
}

func schedule(tasks chan<- JobRequest, j JobRequest) {
	go func() {
		wg.Add(1)
		select {
		case tasks <- j:
		}

		wg.Done()
	}()
}

func startWorker(cli *client.Client, reqs <-chan JobRequest, quit chan<- struct{}, gpu string) {
	for t := range reqs {
		log.Printf("[%s] worker running task\n", time.Now())
		if err := t.Run(context.TODO(), cli, []string{gpu}); err != nil {
			log.Printf("error running task: %s", err)
		}
	}
	quit <- struct{}{}
}
