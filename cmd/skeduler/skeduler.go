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
		Commands: []*cli.Command{
			{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Lists all experiments",
				Action: func(c *cli.Context) error {
					// load file
					cfg, err := config.DecodeFromFile[conf](c.String("config"))
					if err != nil {
						panic(err)
					}
					fmt.Printf("%v\n", cfg)

					listExperiments(cfg.Host, cfg.Token)

					return nil
				},
			},
			{
				Name:      "show",
				Aliases:   []string{"s"},
				Usage:     "Shows an experiments",
				ArgsUsage: "<id>",
				Action: func(c *cli.Context) error {
					cfg, err := config.DecodeFromFile[conf](c.String("config"))
					if err != nil {
						panic(err)
					}
					fmt.Printf("%v\n", cfg)

					if c.Args().Len() > 0 {
						showExperiment(cfg.Host, cfg.Token, c.Args().Get(0))
					} else {
						fmt.Println("Experiment ID not specified")
					}

					return nil
				},
			},
			{
				Name:      "enque",
				Aliases:   []string{"e"},
				Usage:     "Enques an experiment",
				ArgsUsage: "<filename>",
				Action: func(c *cli.Context) error {
					cfg, err := config.DecodeFromFile[conf](c.String("config"))
					if err != nil {
						panic(err)
					}
					fmt.Printf("%v\n", cfg)

					if c.Args().Len() > 0 {
						enqueExperiment(cfg.Host, cfg.Token, c.Args().Get(0))
					} else {
						fmt.Println("Input experiment file not specified")
					}

					return nil
				},
			},
			{
				Name:      "update",
				Aliases:   []string{"u"},
				Usage:     "Updates an experiment",
				ArgsUsage: "<filename>",
				Action: func(c *cli.Context) error {
					cfg, err := config.DecodeFromFile[conf](c.String("config"))
					if err != nil {
						panic(err)
					}
					fmt.Printf("%v\n", cfg)

					if c.Args().Len() > 0 {
						updateExperiment(cfg.Host, cfg.Token, c.Args().Get(0))
					} else {
						fmt.Println("Input experiment file not specified")
					}

					return nil
				},
			},
			{
				Name:      "logs",
				Aliases:   []string{"l"},
				Usage:     "Shows an experiment's logs",
				ArgsUsage: "<id>",
				Action: func(c *cli.Context) error {
					cfg, err := config.DecodeFromFile[conf](c.String("config"))
					if err != nil {
						panic(err)
					}
					fmt.Printf("%v\n", cfg)

					if c.Args().Len() > 1 {
						showLogs(cfg.Host, cfg.Token, c.Args().Get(1))
					} else {
						fmt.Println("Experiment ID not specified")
					}

					return nil
				},
			},
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
