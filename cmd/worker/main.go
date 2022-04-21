package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

var (
	flagConfig = flag.String("config", "config.yml", "Configuration file path")
)

type QueueConfig struct {
	GPUs []string `yaml:"gpus"`
}

type conf struct {
	Host   string        `yaml:"host"`
	Queues []QueueConfig `yaml:"queues"`
}

func main() {
	flag.Parse()

	cfg, err := config.DecodeFromFile[conf](*flagConfig)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded worker configuration: %+v\n", cfg)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	tasks := make(chan jobs.Job, len(cfg.Queues))
	waitWkEnd := make(chan struct{}, len(cfg.Queues))
	for i, wConf := range cfg.Queues {
		a := worker{
			id:   i,
			cli:  cli,
			reqs: tasks,
			quit: waitWkEnd,
			gpus: wConf.GPUs,
			host: cfg.Host,
		}
		go a.start()
	}

	// puller close
	closing := make(chan struct{}, 1)
	go puller(tasks, closing, cfg.Host)

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
	for range cfg.Queues {
		<-waitWkEnd
	}

	log.Printf("shutdown!\n")
}
