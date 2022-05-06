package database

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type Database interface {
	// FetchJob gets a job and updates its status to "RUNNING"
	FetchJob(context.Context) (*jobs.Job, error)

	GetAll(ctx context.Context) ([]jobs.Job, error)
	GetById(context.Context, uuid.UUID) (*jobs.Job, error)

	Insert(context.Context, InsertParams) (*jobs.Job, error)
	Update(context.Context, UpdateParams) (*jobs.Job, error)

	Close() error
}

type InsertParams struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Docker      jobs.Docker `json:"docker"`
	Metadata    interface{} `json:"metadata"`
}

type UpdateParams struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Metadata    interface{}
	Status      jobs.JobStatus `json:"status"`
}
