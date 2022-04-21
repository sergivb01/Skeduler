package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func main() {
	conf, err := configFromFile("config.yml")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded worker configuration: %+v\n", conf)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	tasks := make(chan jobs.Job, len(conf.Queues))
	waitWkEnd := make(chan struct{}, len(conf.Queues))
	for i, wConf := range conf.Queues {
		a := worker{
			id:   i,
			cli:  cli,
			reqs: tasks,
			quit: waitWkEnd,
			gpus: wConf.GPUs,
			host: conf.Host,
		}
		go a.start()
	}

	// puller close
	closing := make(chan struct{}, 1)
	go puller(tasks, closing, conf.Host)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c

	log.Printf("starting gracefull shutdown. Waiting for all pending tasks to finish\n")

	// close workers, they won't work anymore
	close(tasks)

	// notify to close
	for i := 0; i < len(closing); i++ {
		closing <- struct{}{}
	}

	// wait for workers to close
	for range conf.Queues {
		<-waitWkEnd
	}

	log.Printf("shutdown!\n")
}
