package main

import (
	"context"
	"fmt"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func main() {
	db, err := database.NewPostgres("postgres://skeduler:skeduler1234@localhost:5432/skeduler")
	if err != nil {
		panic(err)
	}

	job := &jobs.Job{
		Name:        "test example",
		Description: "test example description",
		Docker: jobs.Docker{
			Image:   "nvidia/cuda:11.0-base",
			Command: "nvidia-smi",
			Environment: map[string]interface{}{
				"TEST": 1234,
			},
		},
	}
	err = db.PutJob(context.TODO(), job)
	if err != nil {
		panic(err)
	}

	job2, err := db.FetchJob(context.TODO())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Job1: %+v\n", job)
	fmt.Printf("Job2: %+v\n", job2)
}
