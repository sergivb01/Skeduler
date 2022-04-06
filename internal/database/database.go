package database

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type Database interface {
	// FetchJob gets a job and updates its status to "RUNNING"
	FetchJob(context.Context) (*jobs.Job, error)

	// GetJobById returns a job given an id
	GetJobById(context.Context, uuid.UUID) (*jobs.Job, error)

	InsertJob(context.Context, *jobs.Job) error
	Update(context.Context, *jobs.Job) error

	Close() error
}
