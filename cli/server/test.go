package main

import (
	"context"
	"fmt"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
)

func main() {
	db, err := database.NewPostgres("postgres://skeduler:skeduler1234@localhost:5432/skeduler")
	if err != nil {
		panic(err)
	}

	job, err := db.FetchJob(context.TODO())
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", job)
}
