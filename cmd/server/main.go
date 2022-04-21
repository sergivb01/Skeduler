package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
)

var (
	confFlag = flag.String("config", "config.yml", "Configuration file path")
)

func main() {
	flag.Parse()

	conf, err := configFromFile(*confFlag)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded server configuration: %+v\n", conf)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	db, err := database.NewPostgres(context.Background(), "postgres://skeduler:skeduler1234@localhost:5432/skeduler")
	// db, err := database.NewSqlite("database.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 2 per http i el puller
	closing := make(chan struct{}, 1)
	waitClose := make(chan struct{}, 1)

	go func() {
		if err := startHttp(closing, conf.Http, db, waitClose); err != nil {
			log.Printf("error server: %s\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c

	log.Printf("starting gracefull shutdown. Waiting for all pending tasks to finish\n")

	// notify to close http & puller
	for i := 0; i < len(closing); i++ {
		closing <- struct{}{}
	}

	// wait http and puller
	for i := 0; i < len(waitClose); i++ {
		<-waitClose
	}

	log.Printf("shutdown!\n")
}
