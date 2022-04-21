package main

import (
	"flag"
	"log"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
)

var (
	flagConfig = flag.String("config", "config.yml", "Configuration file path")
)

type conf struct {
	Host  string `json:"host" yaml:"host"`
	Token string `json:"token" yaml:"token"`
}

func main() {
	flag.Parse()

	cfg, err := config.DecodeFromFile[conf](*flagConfig)
	if err != nil {
		panic(err)
	}

	log.Printf("%v", cfg)
}
