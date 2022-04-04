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
	rows, err := p.db.Query(ctx, `UPDATE jobs SET status = 'RUNNING'::job_status, updated_at = now() WHERE id = (
	    SELECT id
	        FROM jobs
	        WHERE status = 'ENQUEUED'::job_status
	        ORDER BY created_at
	        FOR UPDATE SKIP LOCKED
	        LIMIT 1)
    `)
	if err != nil {
		return nil, fmt.Errorf("querying for job: %w\n", err)
	}
	defer rows.Close()

	var job jobs.Job
	if err := pgxscan.ScanOne(&job, rows); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scaning into struct: %w", err)
	}

	return &job, nil
}

func (p postgresDb) GetJobById(ctx context.Context, uuid uuid.UUID) (jobs.Job, error) {
	// TODO implement me
	panic("implement me")
}

func (p postgresDb) PutJob(ctx context.Context, job jobs.Job) (uuid.UUID, error) {
	// TODO implement me
	panic("implement me")
}

func (p postgresDb) Update(ctx context.Context, uuid uuid.UUID) error {
	// TODO implement me
	panic("implement me")
}

func (p postgresDb) Close() error {
	// TODO implement me
	panic("implement me")
}
