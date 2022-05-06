package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gofrs/uuid"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/config"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
	"log"
	"os"
	"strings"
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
 * fer un tail dels logs (sortida per stderr) en temps real
*/

func main() {
	flag.Parse()

	cfg, err := config.DecodeFromFile[conf](*flagConfig)
	if err != nil {
		panic(err)
	}

	log.Printf("%v", cfg)

	for {
		handleOption(showMenu(), cfg.Host, cfg.Token)
	}

}

func showMenu() int {
	fmt.Println("[1] - List all experiments")
	fmt.Println("[2] - Consult an experiments")
	fmt.Println("[3] - Encue a new exeperiment")
	fmt.Println("[4] - Update exeperiment")
	fmt.Println("[5] - View an experiment logs")

	return readInput(5)

}

func readInput(options int) int {
	ret := -1

	for ret < 1 || ret > options {
		fmt.Printf("Enter an option (1 .. %d): ", options)
		fmt.Scan(&ret)
	}

	return ret
}

func readID() uuid.UUID {
	ret := ""

	fmt.Printf("Enter an experiment ID: ")
	fmt.Scan(&ret)

	return uuid.FromStringOrNil(ret)
}

func readNewExperiment() database.InsertParams {
	in := bufio.NewReader(os.Stdin)

	var ret database.InsertParams
	var docker jobs.Docker

	fmt.Printf("Name: ")
	fmt.Scanln(&ret.Name)

	fmt.Printf("Description: ")
	desc, _ := in.ReadString('\n')
	ret.Description = strings.TrimSuffix(desc, "\n")

	fmt.Printf("Image: ")
	fmt.Scanln(&docker.Image)

	fmt.Printf("Command: ")
	command, _ := in.ReadString('\n')
	docker.Command = strings.TrimSuffix(command, "\n")

	// TODO: metadata & enviroment

	ret.Docker = docker

	return ret
}

func readUpdateExperiment() database.UpdateParams {
	var ret database.UpdateParams

	ret.Id = readID()

	// TODO: metadata & enviroment
	fmt.Printf("Name: ")
	fmt.Scanln(&ret.Name)
	fmt.Printf("Description: ")
	fmt.Scanln(&ret.Description)
	fmt.Printf("Status: ")
	fmt.Scanln(&ret.Status)

	return ret
}

func handleOption(option int, host string, token string) {
	switch option {
	case 1:
		ret, err := getJobs(context.TODO(), host, token)
		if err == nil {
			buf := bytes.NewBufferString("")
			_ = json.NewEncoder(buf).Encode(&ret)
			fmt.Println(PrettyString(buf.String()))
		} else {
			fmt.Println(err.Error())
		}
	case 2:
		ret, err := getJob(context.TODO(), host, token, readID())
		if err == nil {
			buf := bytes.NewBufferString("")
			_ = json.NewEncoder(buf).Encode(&ret)
			fmt.Println(PrettyString(buf.String()))
		} else {
			fmt.Println(err.Error())
		}
	case 3:
		j := readNewExperiment()
		ret, err := newJob(context.TODO(), host, token, j)
		if err == nil {
			buf := bytes.NewBufferString("")
			_ = json.NewEncoder(buf).Encode(&ret)
			fmt.Println(PrettyString(buf.String()))
		} else {
			fmt.Println(err.Error())
		}
	case 4:
		j := readUpdateExperiment()
		ret, err := jobUpdate(context.TODO(), host, token, j)
		if err == nil {
			buf := bytes.NewBufferString("")
			_ = json.NewEncoder(buf).Encode(&ret)
			fmt.Println(PrettyString(buf.String()))
		} else {
			fmt.Println(err.Error())
		}
	case 5:
		ret, err := getLogs(context.TODO(), host, token, readID())
		if err == nil {
			fmt.Println(ret)
		} else {
			fmt.Println(err.Error())
		}
	}
}

func PrettyString(str string) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(str), "", "    "); err != nil {
		return ""
	}
	return prettyJSON.String()
}
