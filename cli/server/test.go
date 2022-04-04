package main

// import (
// 	"context"
// 	"fmt"
//
// 	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
// 	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
// )
//
// func main() {
// 	db, err := database.NewSqlite("database.db")
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	j := &jobs.Job{
// 		Name:        "testing 123",
// 		Description: "descriptionn",
// 		Docker: jobs.Docker{
// 			Image:   "nvidia/cuda:11.0-base",
// 			Command: "nvidia-smi",
// 			Environment: map[string]interface{}{
// 				"TEST": 123,
// 				"ABC":  "awdawd",
// 			},
// 		},
// 	}
//
// 	res, err := db.PutJob(context.Background(), *j)
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("result from insert = %+v\n", res)
//
// 	j2, err := db.GetJob(context.Background())
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("result from get = %+v\n", j2)
//
// 	if err := db.UpdateStatus(context.Background(), j2.ID, jobs.Done); err != nil {
// 		panic(err)
// 	}
// }
