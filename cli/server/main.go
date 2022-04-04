package main

//
// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"
//
// 	"github.com/docker/docker/client"
// 	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
// 	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
// )
//
// func main() {
// 	conf, err := FromFile("config.yml")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%+v\n", conf)
//
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer cli.Close()
//
// 	db, err := database.NewSqlite("database.db")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer db.Close()
//
// 	tasks := make(chan jobs.Job, len(conf.Queues))
// 	waitWkEnd := make(chan struct{}, len(conf.Queues))
// 	for i, q := range conf.Queues {
// 		go startWorker(cli, tasks, waitWkEnd, i, q.GPUs)
// 	}
//
// 	closing := make(chan struct{}, 2)
// 	go func() {
// 		err := startHttp(closing, conf.Http)
// 		if err != nil {
// 			log.Printf("error server: %s\n", err)
// 		}
// 	}()
//
// 	go puller(tasks, closing, db)
//
// 	c := make(chan os.Signal, 1)
// 	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
// 	<-c
//
// 	log.Printf("starting gracefull shutdown. Waiting for all pending tasks to finish\n")
//
// 	// close workers, they won't work anymore
// 	close(tasks)
//
// 	// notify to close http & puller
// 	for i := 0; i < len(closing); i++ {
// 		closing <- struct{}{}
// 	}
//
// 	// wait http and puller
// 	for i := 0; i < len(closing); i++ {
// 		<-closing
// 	}
//
// 	// wait for workers to close
// 	for range conf.Queues {
// 		<-waitWkEnd
// 	}
//
// 	log.Printf("shutdown!\n")
// }
//
// func puller(tasks chan<- jobs.Job, closing <-chan struct{}, db database.Database) {
// 	t := time.Tick(time.Second * 3)
// 	for {
// 		select {
// 		case <-closing:
// 			return
// 		case <-t:
// 			if len(tasks) != cap(tasks) {
// 				job, err := db.FetchJob(context.TODO())
// 				if err != nil {
// 					log.Printf("error pulling: %v\n", err)
// 				}
// 				if job == nil {
// 					continue
// 				}
// 				tasks <- *job
// 			}
//
// 			break
// 		}
// 	}
// }
//
// func startWorker(cli *client.Client, reqs <-chan jobs.Job, quit chan<- struct{}, id int, gpus []string) {
// 	for t := range reqs {
// 		log.Printf("[%d] worker running task %+v at %s\n", id, t, time.Now())
// 		if err := t.Run(context.TODO(), cli, gpus); err != nil {
// 			log.Printf("error running task: %s", err)
// 		}
// 	}
// 	quit <- struct{}{}
// }
