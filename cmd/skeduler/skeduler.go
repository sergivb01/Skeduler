package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/urfave/cli/v2"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"io/ioutil"
	"log"
	"os"
)

type conf struct {
	Host  string `json:"host" yaml:"host"`
	Token string `json:"token" yaml:"token"`
}

func main() {

	app := &cli.App{
		Name:  "skeduler",
		Usage: "Encuador d'experiments amb Docker",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"f"},
				Value:   "config_client.json",
				Usage:   "Configuration file path",
			},
		},
		Action: func(c *cli.Context) error {
			cfg, err := config.DecodeFromFile[conf](c.String("config"))

			if err != nil {
				panic(err)
			}

			fmt.Printf("%v\n", cfg)

			if c.Args().Get(0) == "experiments" { // show all experiments
				listExperiments(cfg.Host, cfg.Token)
			} else if c.Args().Get(0) == "experiment" { // show experiment
				if c.Args().Len() > 1 {
					showExperiment(cfg.Host, cfg.Token, c.Args().Get(1))
				} else {
					fmt.Println("Experiment ID not specified")
				}
			} else if c.Args().Get(0) == "enque" { // enque new experiment
				if c.Args().Len() > 1 {
					enqueExperiment(cfg.Host, cfg.Token, c.Args().Get(1))
				} else {
					fmt.Println("Input experiment file not specified")
				}
			} else if c.Args().Get(0) == "update" { // update experiment
				if c.Args().Len() > 1 {
					updateExperiment(cfg.Host, cfg.Token, c.Args().Get(1))
				} else {
					fmt.Println("Input update file not specified")
				}
			} else if c.Args().Get(0) == "logs" { // show experiment logs
				if c.Args().Len() > 1 {
					if c.Bool("live") {

					} else {
						showLogs(cfg.Host, cfg.Token, c.Args().Get(1))
					}
				} else {
					fmt.Println("Experiment ID not specified")
				}
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func listExperiments(host string, token string) {
	ret, err := getJobs(context.TODO(), host, token)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		buf := bytes.NewBufferString("")
		_ = json.NewEncoder(buf).Encode(&ret)
		fmt.Println(PrettyString(buf.String()))
	}
}

func showExperiment(host string, token string, id string) {
	jobId, _ := uuid.FromString(id)
	ret, err := getJob(context.TODO(), host, token, jobId)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		buf := bytes.NewBufferString("")
		_ = json.NewEncoder(buf).Encode(&ret)
		fmt.Println(PrettyString(buf.String()))
	}
}

func enqueExperiment(host string, token string, file string) {
	jsonFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	ret, err := newJob(context.TODO(), host, token, string(byteValue))
	if err != nil {
		fmt.Println(err.Error())
	} else {
		buf := bytes.NewBufferString("")
		_ = json.NewEncoder(buf).Encode(&ret)
		fmt.Println(PrettyString(buf.String()))
	}

}

func updateExperiment(host string, token string, file string) {
	jsonFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	buf := bytes.NewBufferString(string(byteValue))
	_ = json.NewEncoder(buf).Encode(&buf)

	var job database.UpdateParams
	_ = json.NewDecoder(buf).Decode(&job)

	ret, err := jobUpdate(context.TODO(), host, token, job)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		buf := bytes.NewBufferString("")
		_ = json.NewEncoder(buf).Encode(&ret)
		fmt.Println(PrettyString(buf.String()))
	}
}

func showLogs(host string, token string, id string) {
	jobId, _ := uuid.FromString(id)
	ret, err := getLogs(context.TODO(), host, token, jobId)

	if err == nil {
		fmt.Println(ret)
	} else {
		fmt.Println(err.Error())
	}
}

func PrettyString(str string) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(str), "", "    "); err != nil {
		return ""
	}
	return prettyJSON.String()
}
