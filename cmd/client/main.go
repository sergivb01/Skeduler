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

/**
TODO:
 * llistat d'experiments (poder filtrar per encuats/corrent/finalitzats/cancelats) (falta backend)
 * consultar un experiment concret
 * crear nou experiment
 * actualitzar experiment
 _
 * consultar TOTS els logs d'experiment
 * fer un tail dels logs (sortida per stderr) en temps real
*/

func main() {
	flag.Parse()

	// dir, err := os.UserHomeDir()
	// if err != nil {
	// 	panic(err)
	// }
	// path := fmt.Sprintf("%s/.skeduler.json", dir)

	cfg, err := config.DecodeFromFile[conf](*flagConfig)
	if err != nil {
		panic(err)
	}

	log.Printf("%v", cfg)

}
