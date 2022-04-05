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

func NewPostgres(uri string) (*postgresDb, error) {
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

	db, err := pgxpool.ConnectConfig(context.TODO(), pgxconfig)
	if err != nil {
		return nil, fmt.Errorf("connecting postgres: %w", err)
	}

	if err := db.Ping(context.TODO()); err != nil {
		return nil, fmt.Errorf("pining postgres: %w", err)
	}

	return &postgresDb{db: db}, nil
}

func (p postgresDb) FetchJob(ctx context.Context) (*jobs.Job, error) {
	rows, err := p.db.Query(ctx, `UPDATE jobs
		SET status     = 'RUNNING'::job_status,
		    updated_at = now()
		WHERE id = (
		    SELECT id
		    FROM jobs
		    WHERE status = 'ENQUEUED'::job_status
		    ORDER BY created_at
		        FOR UPDATE SKIP LOCKED
		    LIMIT 1)
		RETURNING id, name, description, docker_image AS "docker_embedded.docker_image", docker_command AS "docker_embedded.docker_command",
		    docker_environment AS "docker_embedded.docker_environment", created_at, updated_at, status
    `)
	if err != nil {
		return nil, fmt.Errorf("querying for job: %w\n", err)
	}
	defer rows.Close()

	var job jobs.Job
	if err := jobFromRows(rows, &job); err != nil {
		return nil, fmt.Errorf("scanning rows from fetch: %w", err)
	}

	return &job, nil
}

func (p postgresDb) GetJobById(ctx context.Context, id uuid.UUID) (*jobs.Job, error) {
	rows, err := p.db.Query(ctx, `SELECT id, name, description, docker_image AS "docker_embedded.docker_image", docker_command AS "docker_embedded.docker_command",
       docker_environment AS "docker_embedded.docker_environment", created_at, updated_at, status
       FROM jobs
       WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("selecting job by id: %w", err)
	}
	defer rows.Close()

	var job jobs.Job
	if err := jobFromRows(rows, &job); err != nil {
		return nil, fmt.Errorf("scanning rows from select: %w", err)
	}

	return &job, nil
}

func (p postgresDb) PutJob(ctx context.Context, job *jobs.Job) error {
	rows, err := p.db.Query(ctx, `INSERT INTO jobs (name, description, docker_image, docker_command, docker_environment) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		job.Name, job.Description, job.Docker.Image, job.Docker.Command, job.Docker.Environment)
	if err != nil {
		return fmt.Errorf("inserting new job: %w", err)
	}

	return jobFromRows(rows, job)
}

func (p postgresDb) Update(ctx context.Context, job *jobs.Job) error {
	// TODO(@sergivb01): use res
	_, err := p.db.Exec(ctx, `UPDATE jobs SET name = $2, description = $3, docker_image = $4, docker_command = $5, docker_environment = $6, updated_at = now() WHERE id = $1`,
		job.ID, job.Name, job.Description, job.Docker.Image, job.Docker.Command, job.Docker.Environment)
	if err != nil {
		return fmt.Errorf("updating job: %w", err)
	}
	return nil
}

func (p postgresDb) Close() error {
	p.db.Close()
	return nil
}

func jobFromRows(rows pgx.Rows, job *jobs.Job) error {
	if err := pgxscan.ScanOne(job, rows); err != nil && errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("scaning into struct: %w", err)
	}
	return nil
}
