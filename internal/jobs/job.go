package jobs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/gofrs/uuid"
)

type Docker struct {
	Image       string                 `json:"image" db:"docker_image"`
	Command     string                 `json:"command" db:"docker_command"`
	Environment map[string]interface{} `json:"environment" db:"docker_environment"`
}

type JobStatus string

const (
	Enqueued  JobStatus = "ENQUEUED"
	Running   JobStatus = "RUNNING"
	Finished  JobStatus = "FINISHED"
	Cancelled JobStatus = "CANCELLED"
)

type Job struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Docker      Docker      `json:"docker" db:"docker_embedded"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
	Status      JobStatus   `json:"status" db:"status"`
	Metadata    interface{} `json:"metadata" db:"metadata"`
}

const MagicEnd = "_#$#$#$<END>#$#$#$_"

func NewFromFile(filename string) (*Job, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	var r Job
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	return &r, nil
}
