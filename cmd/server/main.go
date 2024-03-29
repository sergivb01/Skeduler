package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
)

// httpConfig is the configuration for the HTTP server.
type httpConfig struct {
	Listen      string        `yaml:"listen" json:"listen"`
	ReadTimeout time.Duration `yaml:"read_timeout" json:"readTimeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout" json:"idleTimeout"`
}

// config is the configuration for the server.
type conf struct {
	Database string         `yaml:"database" json:"database"`
	Http     httpConfig     `yaml:"http" json:"http"`
	Tokens   []string       `yaml:"tokens" json:"tokens"`
	Telegram telegramClient `yaml:"telegram" json:"telegram"`
}

var (
	flagConfig = flag.String("config", "config.yml", "Configuration file path")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.DecodeFromFile[conf](*flagConfig)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded server configuration: %+v\n", cfg)

	// Connect to database
	db, err := database.NewPostgres(context.Background(), cfg.Database)
	// db, err := database.NewSqlite("database.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 2 per http i el puller
	closing := make(chan struct{}, 1)
	waitClose := make(chan struct{}, 1)

	// Start HTTP server
	go func() {
		if err := startHttp(closing, *cfg, db, waitClose); err != nil {
			log.Printf("error server: %s\n", err)
		}
	}()

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c

	log.Printf("starting gracefull shutdown. Waiting for all pending tasks to finish\n")

	// Stop HTTP server
	for i := 0; i < len(closing); i++ {
		closing <- struct{}{}
	}

	// Wait for all pending tasks to finish
	for i := 0; i < len(waitClose); i++ {
		<-waitClose
	}

	log.Printf("shutdown!\n")
}
