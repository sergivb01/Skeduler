package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type sqliteDb struct {
	db *sql.DB
}

func (s sqliteDb) GetById(ctx context.Context, id uuid.UUID) (*jobs.Job, error) {
	var job jobs.Job
	err := s.runQuery(ctx, &job, "SELECT id, name, description, docker_image, docker_cmd, docker_env, created_at, updated_at, status, metadata FROM jobs WHERE id = ?", id)

	if err != nil {
		return nil, fmt.Errorf("getting job by id: %w", err)
	}

	return &job, nil
}

func (s sqliteDb) Insert(ctx context.Context, params InsertParams) (*jobs.Job, error) {
	env := envToString(params.Docker.Environment)
	b, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata into json: %w", err)
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("generating uuid: %w", err)
	}

	job := &jobs.Job{}
	err = s.runQuery(ctx, job, `INSERT INTO jobs (id, name, description, docker_image, docker_cmd, docker_env, metadata) VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, 'name', description, docker_image, docker_cmd, docker_env, datetime(created_at,'unixepoch'), datetime(updated_at,'unixepoch'), status, metadata`,
		id, params.Name, params.Description, params.Docker.Image, params.Docker.Command, env, string(b))

	if err != nil {
		return nil, fmt.Errorf("inserting job: %w", err)
	}

	return job, nil
}

func (s sqliteDb) Update(ctx context.Context, params UpdateParams) (*jobs.Job, error) {
	b, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata into json: %w", err)
	}

	job := &jobs.Job{}
	err = s.runQuery(ctx, job, `UPDATE jobs SET name = ?, description = ?, metadata = ?, status = ?, updated_at = strftime('%s', 'now')
			WHERE id = ?
			RETURNING id, 'name', description, docker_image, docker_cmd, docker_env, datetime(created_at,'unixepoch'), datetime(updated_at,'unixepoch'), status, metadata`,
		params.Name, params.Description, string(b), params.Status, job.ID)

	if err != nil {
		return nil, fmt.Errorf("updating job: %w", err)
	}

	return job, nil
}

const (
	timeFormat = "2006-01-02 15:04:05"
	envSplit   = "_#&#_"
)

var _ Database = sqliteDb{}

func NewSqlite(filename string) (*sqliteDb, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	return &sqliteDb{db: db}, nil
}

func (s sqliteDb) FetchJob(ctx context.Context) (*jobs.Job, error) {
	var job jobs.Job
	err := s.runQuery(ctx, &job, `UPDATE jobs SET status = 'RUNNING', updated_at = strftime('%s', 'now')
		WHERE rowid = (
		    SELECT min(rowid) FROM jobs WHERE status = 'ENQUEUED'
	    )
	    RETURNING id, 'name', description, docker_image, docker_cmd, docker_env, datetime(created_at,'unixepoch'), datetime(updated_at,'unixepoch'), status, metadata
    `)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching job: %w", err)
	}

	return &job, nil
}

func (s sqliteDb) GetAll(ctx context.Context) ([]jobs.Job, error) {
	panic("GetAll in sqlite not implemented")
}

func (s sqliteDb) runQuery(ctx context.Context, job *jobs.Job, query string, args ...interface{}) error {
	row := s.db.QueryRowContext(ctx, query, args...)
	if err := row.Err(); err != nil {
		return fmt.Errorf("fetching job: %w", err)
	}

	var (
		createdAt string
		updatedAt string
		dockerEnv string
		meta      string
	)

	err := row.Scan(&job.ID, &job.Name, &job.Description, &job.Docker.Image, &job.Docker.Command, &dockerEnv, &createdAt, &updatedAt, &job.Status, &meta)
	if err != nil {
		return fmt.Errorf("scanning result to struct: %w", err)
	}

	job.Docker.Environment = stringToEnv(dockerEnv)
	job.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	job.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)

	if err := json.Unmarshal([]byte(meta), &job.Metadata); err != nil {
		return fmt.Errorf("unmarshaling metadata: %w", err)
	}

	return nil
}

func (s sqliteDb) Close() error {
	return s.db.Close()
}

func envToString(env map[string]interface{}) string {
	var sb strings.Builder
	for k, v := range env {
		sb.WriteString(k)
		sb.WriteRune('=')
		sb.WriteString(fmt.Sprintf("%v", v))
		sb.WriteString(envSplit)
	}
	return strings.TrimSuffix(sb.String(), envSplit)
}

func stringToEnv(val string) map[string]interface{} {
	env := make(map[string]interface{})
	for _, kv := range strings.Split(val, envSplit) {
		parts := strings.SplitN(kv, "=", 2)
		env[parts[0]] = parts[1]
	}

	return env
}
