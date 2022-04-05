package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	pgtypeuuid "github.com/jackc/pgtype/ext/gofrs-uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type postgresDb struct {
	db *pgxpool.Pool
}

var _ Database = &postgresDb{}

func NewPostgres(ctx context.Context, uri string) (*postgresDb, error) {
	pgxconfig, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, fmt.Errorf("parsing db uri: %w", err)
	}
	pgxconfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.ConnInfo().RegisterDataType(pgtype.DataType{
			Value: &pgtypeuuid.UUID{},
			Name:  "uuid",
			OID:   pgtype.UUIDOID,
		})
		enumType := pgtype.NewEnumType("job_status", []string{"ENQUEUED", "RUNNING", "FINISHED", "CANCELLED"})
		conn.ConnInfo().RegisterDataType(pgtype.DataType{
			Value: enumType,
			Name:  "job_status",
		})
		return nil
	}

	db, err := pgxpool.ConnectConfig(ctx, pgxconfig)
	if err != nil {
		return nil, fmt.Errorf("connecting postgres: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pining postgres: %w", err)
	}

	return &postgresDb{db: db}, nil
}

func (p postgresDb) FetchJob(ctx context.Context) (*jobs.Job, error) {
	var job jobs.Job
	err := p.runQuery(ctx, &job, `UPDATE jobs
		SET status     = 'RUNNING'::job_status,
		    updated_at = current_timestamp
		WHERE id = (
		    SELECT id
		    FROM jobs
		    WHERE status = 'ENQUEUED'::job_status
		    ORDER BY created_at
		        FOR UPDATE SKIP LOCKED
		    LIMIT 1)
		RETURNING id, name, description, docker_image AS "docker_embedded.docker_image", docker_command AS "docker_embedded.docker_command",
		    docker_environment AS "docker_embedded.docker_environment", created_at, updated_at, status, metadata
    `)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching job: %w", err)
	}

	return &job, nil
}

func (p postgresDb) GetJobById(ctx context.Context, id uuid.UUID) (*jobs.Job, error) {
	var job jobs.Job
	err := p.runQuery(ctx, &job, `SELECT id, name, description, docker_image AS "docker_embedded.docker_image", docker_command AS "docker_embedded.docker_command",
       docker_environment AS "docker_embedded.docker_environment", created_at, updated_at, status, metadata
       FROM jobs
       WHERE id = $1`, id)

	if err != nil {
		return nil, fmt.Errorf("getting job by id: %w", err)
	}

	return &job, nil
}

func (p postgresDb) InsertJob(ctx context.Context, job *jobs.Job) error {
	err := p.runQuery(ctx, job, `INSERT INTO jobs (name, description, docker_image, docker_command, docker_environment, metadata) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, docker_image AS "docker_embedded.docker_image", docker_command AS "docker_embedded.docker_command",
       		docker_environment AS "docker_embedded.docker_environment", created_at, updated_at, status, metadata`,
		job.Name, job.Description, job.Docker.Image, job.Docker.Command, job.Docker.Environment, job.Metadata)

	if err != nil {
		return fmt.Errorf("inserting job: %w", err)
	}

	return nil
}

func (p postgresDb) Update(ctx context.Context, job *jobs.Job) error {
	err := p.runQuery(ctx, job, `UPDATE jobs
		SET name = $2, description = $3, docker_image = $4, docker_command = $5, docker_environment = $6, updated_at = current_timestamp, status = $7, metadata = $8
		WHERE id = $1
		RETURNING id, name, description, docker_image AS "docker_embedded.docker_image", docker_command AS "docker_embedded.docker_command",
       		docker_environment AS "docker_embedded.docker_environment", created_at, updated_at, status, metadata`,
		job.ID, job.Name, job.Description, job.Docker.Image, job.Docker.Command, job.Docker.Environment, job.Status, job.Metadata)
	if err != nil {
		return fmt.Errorf("updating job: %w", err)
	}
	return nil
}

func (p postgresDb) Close() error {
	p.db.Close()
	return nil
}

func (p postgresDb) runQuery(ctx context.Context, job *jobs.Job, query string, args ...interface{}) error {
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("running query: %w", err)
	}
	defer rows.Close()

	if err := jobFromRows(rows, job); err != nil {
		return fmt.Errorf("scanning rows: %w", err)
	}
	return nil
}

func jobFromRows(rows pgx.Rows, job *jobs.Job) error {
	if err := pgxscan.ScanOne(job, rows); err != nil && errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("scaning into struct: %w", err)
	}
	return nil
}
