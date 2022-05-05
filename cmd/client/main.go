package main

import (
	"context"
	"flag"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
	"log"
)

var (
	flagConfig = flag.String("config", "config.yml", "Configuration file path")
)

type conf struct {
	Host  string `json:"host" yaml:"host"`
	Token string `json:"token" yaml:"token"`
}

/**
TODO:
 * llistat d'experiments (poder filtrar per encuats/corrent/finalitzat/cancelats) (falta backend)
 * consultar un experiment concret
 * crear nou experiment
 * actualitzar experiment
 _
 * consultar TOTS els logs d'experiment
 * fer un tail dels logs (sortida per stderr) en temps real
*/

func main() {
	flag.Parse()

	cfg, err := config.DecodeFromFile[conf](*flagConfig)
	if err != nil {
		panic(err)
	}

	log.Printf("%v", cfg)

	var doc jobs.Docker
	doc.Image = "hello-world"
	doc.Command = "docker run hello-world"
	doc.Environment = map[string]interface{}{}

	var prova database.InsertParams
	prova.Name = "prova1"
	prova.Description = "jejejejej"
	prova.Docker = doc

	newJob(context.TODO(), cfg.Host, cfg.Token, prova)
}
