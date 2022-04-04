package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type sqliteDb struct {
	db *sql.DB
}

const timeFormat = "2006-01-02 15:04:05"

var _ Database = sqliteDb{}

func (s sqliteDb) Close() error {
	return s.db.Close()
}

func (s sqliteDb) PutJob(ctx context.Context, j jobs.Job) (jobs.ID, error) {
	env := envToString(j.Docker.Environment)

	res, err := s.db.ExecContext(ctx, "INSERT INTO jobs ('name', description , docker_image, docker_cmd, docker_env) VALUES (?, ?, ?, ?, ?)",
		j.Name, j.Description, j.Docker.Image, j.Docker.Command, env)
	if err != nil {
		return -1, fmt.Errorf("exec insert: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("getting last insert id: %w", err)
	}

	return jobs.ID(id), err
}

func (s sqliteDb) GetJobById(ctx context.Context, id jobs.ID) (jobs.Job, error) {
	// TODO implement me
	panic("implement me")
}

func (s sqliteDb) GetJob(ctx context.Context) (*jobs.Job, error) {

	row := s.db.QueryRowContext(ctx, `UPDATE jobs SET status = 'RUNNING', updated_at = strftime('%s', 'now')
		WHERE rowid = (
		    SELECT min(rowid) FROM jobs WHERE status = 'ENQUEUED'
	    )
	    RETURNING id, 'name', description, docker_image, docker_cmd, docker_env, datetime(created_at,'unixepoch'), datetime(updated_at,'unixepoch'), status
    `)
	if row.Err() != nil {
		return nil, fmt.Errorf("getting task: %w", row.Err())
	}

	var j jobs.Job
	var (
		createdAt string
		updatedAt string
		dockerEnv string
	)

	err := row.Scan(&j.ID, &j.Name, &j.Description, &j.Docker.Image, &j.Docker.Command, &dockerEnv, &createdAt, &updatedAt, &j.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scanning result to struct: %w", err)
	}

	j.Docker.Environment = stringToEnv(dockerEnv)
	j.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	j.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)

	return &j, nil
}

func (s sqliteDb) UpdateStatus(ctx context.Context, id jobs.ID, status jobs.JobStatus) error {
	res, err := s.db.ExecContext(ctx, "UPDATE jobs SET status = ?, updated_at = strftime('%s', 'now') WHERE id = ?", string(status), id)
	if err != nil {
		return fmt.Errorf("updating status for %v: %w", id, err)
	}

	if n, err := res.RowsAffected(); err == nil && n <= 0 {
		return fmt.Errorf("update did not affect a row: %w", err)
	}

	return nil
}

func NewSqlite(filename string) (*sqliteDb, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	return &sqliteDb{db: db}, nil
}
