package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gofrs/uuid"
	"github.com/urfave/cli/v2"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
)

type conf struct {
	Host  string `json:"host" yaml:"host"`
	Token string `json:"token" yaml:"token"`
}

func main() {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	cfg, err := config.DecodeFromFile[conf](fmt.Sprintf("%s/.skeduler.json", dir))
	if err != nil {
		panic(err)
	}

	app := &cli.App{
		Name:  "skeduler",
		Usage: "Encuador d'experiments amb Docker",
		Commands: []*cli.Command{
			{
				Name:    "all",
				Aliases: []string{"a", "ls"},
				Usage:   "Lists all experiments",
				Action: func(c *cli.Context) error {
					return listExperiments(cfg.Host, cfg.Token)
				},
			},
			{
				Name:      "show",
				Aliases:   []string{"s"},
				Usage:     "Shows an experiments",
				ArgsUsage: "<id>",
				Action: func(c *cli.Context) error {
					if c.Args().Len() > 0 {
						return showExperiment(cfg.Host, cfg.Token, c.Args().Get(0))
					}

					fmt.Println("Experiment ID not specified")
					return nil
				},
			},
			{
				Name:      "enqueue",
				Aliases:   []string{"e"},
				Usage:     "Enqueues an experiment",
				ArgsUsage: "<filename>",
				Action: func(c *cli.Context) error {
					if c.Args().Len() > 0 {
						return enqueExperiment(cfg.Host, cfg.Token, c.Args().Get(0))
					}

					fmt.Println("Input experiment file not specified")
					return nil
				},
			},
			{
				Name:      "update",
				Aliases:   []string{"u"},
				Usage:     "Updates an experiment",
				ArgsUsage: "<filename>",
				Action: func(c *cli.Context) error {
					if c.Args().Len() > 0 {
						return updateExperiment(cfg.Host, cfg.Token, c.Args().Get(0))
					}

					fmt.Println("Input experiment file not specified")
					return nil
				},
			},
			{
				Name:      "logs",
				Aliases:   []string{"l"},
				Usage:     "Shows an experiment's logs",
				ArgsUsage: "<id>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "follow",
						Aliases: []string{"f"},
						Usage:   "Follow logs in realtime",
					},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() > 0 {
						if c.Bool("follow") {
							return followLogs(cfg.Host, cfg.Token, c.Args().Get(0))
						}
						return showLogs(cfg.Host, cfg.Token, c.Args().Get(0))
					}

					fmt.Println("Experiment ID not specified")
					return nil
				},
			},
		},
		Authors: []*cli.Author{
			{
				Name:  "Sergi Vos",
				Email: "contacte@sergivos.dev",
			},
			{
				Name:  "Xavier Terés",
				Email: "algo@xavierteres.com",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("error running command: %v", err)
	}

}

func listExperiments(host string, token string) error {
	ret, err := getJobs(context.TODO(), host, token)
	if err != nil {
		return fmt.Errorf("error getting jobs from backend: %w", err)
	}

	buf := bytes.NewBufferString("")
	if err := json.NewEncoder(buf).Encode(&ret); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	fmt.Println(PrettyString(buf.String()))
	return nil
}

func showExperiment(host string, token string, id string) error {
	jobId, _ := uuid.FromString(id)
	ret, err := getJob(context.TODO(), host, token, jobId)
	if err != nil {
		return fmt.Errorf("error getting job from backend: %w", err)
	}

	buf := bytes.NewBufferString("")
	if err := json.NewEncoder(buf).Encode(&ret); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	fmt.Println(PrettyString(buf.String()))
	return nil
}

func enqueExperiment(host, token, fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("error reading specification: %w", err)
	}

	// TODO: no convertir a string
	ret, err := newJob(context.TODO(), host, token, string(b))
	if err != nil {
		return fmt.Errorf("error creating new job: %w", err)
	}

	buf := bytes.NewBufferString("")
	if err := json.NewEncoder(buf).Encode(&ret); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	fmt.Println(PrettyString(buf.String()))
	return nil
}

func updateExperiment(host, token, fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("error reading specification: %w", err)
	}

	// TODO: no convertir a string
	buf := bytes.NewBufferString(string(b))

	var job database.UpdateParams
	if err := json.NewDecoder(buf).Decode(&job); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	ret, err := jobUpdate(context.TODO(), host, token, job)
	if err != nil {
		return fmt.Errorf("error updating job: %w", err)
	}

	buf = bytes.NewBufferString("")
	if err := json.NewEncoder(buf).Encode(&ret); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}
	fmt.Println(PrettyString(buf.String()))

	return nil
}

func showLogs(host string, token string, id string) error {
	jobId, _ := uuid.FromString(id)
	ret, err := getLogs(context.TODO(), host, token, jobId)
	if err != nil {
		return fmt.Errorf("error getting logs: %w", err)
	}

	fmt.Println(ret)
	return nil
}

func followLogs(host, token, id string) error {
	jobId, _ := uuid.FromString(id)
	if err := getLogsFollow(context.TODO(), host, token, jobId, os.Stdout); err != nil {
		return fmt.Errorf("error getting logs live: %w", err)
	}

	return nil
}

func PrettyString(str string) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(str), "", "\t"); err != nil {
		panic(err)
	}
	return prettyJSON.String()
}
