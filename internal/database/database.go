package database

import (
	"context"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type Database interface {
	PutJob(context.Context, jobs.Job) (jobs.ID, error)
	GetJobById(context.Context, jobs.ID) (jobs.Job, error)
	GetJob(context.Context) (*jobs.Job, error)
	UpdateStatus(context.Context, jobs.ID, jobs.JobStatus) error
	Close() error
}
