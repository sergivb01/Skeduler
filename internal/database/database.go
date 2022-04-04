package database

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type Database interface {
	FetchJob(context.Context) (*jobs.Job, error)
	GetJobById(context.Context, uuid.UUID) (jobs.Job, error)

	PutJob(context.Context, jobs.Job) (uuid.UUID, error)
	Update(context.Context, uuid.UUID) error

	Close() error
}
