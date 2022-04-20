package proto

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/nxadm/tail"
	"github.com/nxadm/tail/ratelimiter"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/pkg/backend"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	backend.UnimplementedBackendServer
	db database.Database
}

var _ backend.BackendServer = &server{}

var statusBackToJob = map[backend.JobStatus]jobs.JobStatus{
	backend.JobStatus_ENQUEUED:  jobs.Enqueued,
	backend.JobStatus_RUNNING:   jobs.Running,
	backend.JobStatus_FINISHED:  jobs.Finished,
	backend.JobStatus_CANCELLED: jobs.Cancelled,
}

var statusJobToBack = map[jobs.JobStatus]backend.JobStatus{
	jobs.Enqueued:  backend.JobStatus_ENQUEUED,
	jobs.Running:   backend.JobStatus_RUNNING,
	jobs.Finished:  backend.JobStatus_FINISHED,
	jobs.Cancelled: backend.JobStatus_CANCELLED,
}

func jobToBackend(j jobs.Job) *backend.Job {
	return &backend.Job{
		Id:          j.ID.String(),
		Name:        j.Name,
		Description: j.Description,
		Docker: &backend.Docker{
			Image:       j.Docker.Image,
			Command:     j.Docker.Command,
			Environment: j.Docker.Environment,
		},
		CreatedAt: timestamppb.New(j.CreatedAt),
		UpdatedAt: timestamppb.New(j.UpdatedAt),
		Status:    statusJobToBack[j.Status],
	}
}

func (s *server) Create(ctx context.Context, r *backend.JobCreateRequest) (*backend.Job, error) {
	job, err := s.db.Insert(ctx, database.InsertParams{
		Name:        r.Name,
		Description: r.Description,
		Docker: jobs.Docker{
			Image:       r.Docker.Image,
			Command:     r.Docker.Command,
			Environment: r.Docker.Environment,
		},
	})
	if err != nil {
		return nil, err
	}

	return jobToBackend(*job), nil
}

func (s *server) Get(ctx context.Context, r *backend.GetJobRequest) (*backend.Job, error) {
	id, err := uuid.FromString(r.Id)
	if err != nil {
		return nil, err
	}

	j, err := s.db.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	return jobToBackend(*j), nil
}

func (s *server) Update(ctx context.Context, r *backend.UpdateJobRequest) (*backend.Job, error) {
	id, err := uuid.FromString(r.Id)
	if err != nil {
		return nil, err
	}

	job, err := s.db.Update(ctx, database.UpdateParams{
		Id:          id,
		Name:        r.Name,
		Description: r.Description,
		Status:      statusBackToJob[r.Status],
	})

	return jobToBackend(*job), nil
}

func (s *server) Logs(r *backend.JobLogsRequest, srv backend.Backend_LogsServer) error {
	id, err := uuid.FromString(r.Id)
	if err != nil {
		return err
	}

	// Get a file handle for the file we
	// want to upload
	file, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", id), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	// Allocate a buffer with `chunkSize` as the capacity
	// and length (making a 0 array of the size of `chunkSize`)
	buf := make([]byte, 1024)
	for {
		// put as many bytes as `chunkSize` into the
		// buf array.
		n, err := file.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
		}

		if err := srv.Send(&backend.Chunk{Content: buf[:n]}); err != nil {
			return err
		}
	}

	return nil
}

func (s *server) StreamLogs(r *backend.JobLogsRequest, srv backend.Backend_StreamLogsServer) error {
	id, err := uuid.FromString(r.Id)
	if err != nil {
		return err
	}

	t, err := tail.TailFile(fmt.Sprintf("./logs/%v.log", id), tail.Config{
		ReOpen:      false,
		MustExist:   false, // podem fer el tail abans que existeixi l'execució
		Follow:      true,
		MaxLineSize: 0,
		// TODO: configurar ratelimit
		RateLimiter: ratelimiter.NewLeakyBucket(64, time.Second),
		Logger:      nil,
	})
	if err != nil {
		return err
	}
	defer t.Cleanup()

	for line := range t.Lines {
		if err := line.Err; err != nil {
			// TODO: extreure error
			if err.Error() == "Too much log activity; waiting a second before resuming tailing" {
				continue
			}
			return err
		}

		text := line.Text
		if strings.Contains(text, jobs.MagicEnd) {
			text = strings.TrimSuffix(text, jobs.MagicEnd)
			if err := srv.Send(&backend.Chunk{Content: []byte(text)}); err != nil {
				return err
			}
			return nil
		}

		if err := srv.Send(&backend.Chunk{Content: []byte(text)}); err != nil {
			return err
		}
		break
	}

	return nil
}
