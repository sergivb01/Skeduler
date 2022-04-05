package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

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

	db, err := database.NewPostgres("postgres://skeduler:skeduler1234@localhost:5432/skeduler")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tasks := make(chan jobs.Job, len(conf.Queues))
	waitWkEnd := make(chan struct{}, len(conf.Queues))
	for i, wConf := range conf.Queues {
		a := worker{
			id:   i,
			cli:  cli,
			db:   db,
			reqs: tasks,
			quit: waitWkEnd,
			gpus: wConf.GPUs,
		}
		go a.start()
	}

	closing := make(chan struct{}, 2)
	go func() {
		err := startHttp(closing, conf.Http, db)
		if err != nil {
			log.Printf("error server: %s\n", err)
		}
	}()

	go puller(tasks, closing, db)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c

	log.Printf("starting gracefull shutdown. Waiting for all pending tasks to finish\n")

	// close workers, they won't work anymore
	close(tasks)

	// notify to close http & puller
	for i := 0; i < len(closing); i++ {
		closing <- struct{}{}
	}

	// wait http and puller
	for i := 0; i < len(closing); i++ {
		<-closing
	}

	// wait for workers to close
	for range conf.Queues {
		<-waitWkEnd
	}

	log.Printf("shutdown!\n")
}
