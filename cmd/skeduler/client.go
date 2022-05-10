package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

var errNoJob = errors.New("no job available")

var httpClient = &http.Client{}

func getJobs(ctx context.Context, host, token string) ([]jobs.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/experiments", host), nil)
	req.Header.Set("Authorization", token)

	if err != nil {
		return []jobs.Job{}, fmt.Errorf("creating get request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return []jobs.Job{}, fmt.Errorf("performing get request: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		_, _ = io.Copy(ioutil.Discard, res.Body)
		return []jobs.Job{}, errNoJob

	case http.StatusOK:
		var job []jobs.Job
		_ = json.NewDecoder(res.Body).Decode(&job)
		return job, nil

	default:
	}

	return []jobs.Job{}, fmt.Errorf("server error, recived status code %d and body", res.StatusCode)
}

func newJob(ctx context.Context, host, token string, jobRequest string) (jobs.Job, error) {
	buf := bytes.NewBufferString(jobRequest)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/experiments", host), buf)
	req.Header.Set("Authorization", token)

	if err != nil {
		return jobs.Job{}, fmt.Errorf("creating get request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("performing get request: %w", err)
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		_, _ = io.Copy(ioutil.Discard, res.Body)
		return jobs.Job{}, errNoJob

	case http.StatusOK:
		var job jobs.Job
		_ = json.NewDecoder(res.Body).Decode(&job)
		return job, nil

	default:
	}

	return jobs.Job{}, fmt.Errorf("server error, recived status code %d and body", res.StatusCode)
}

func getJob(ctx context.Context, host, token string, id uuid.UUID) (jobs.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/experiments/%s", host, id.String()), nil)
	req.Header.Set("Authorization", token)

	if err != nil {
		return jobs.Job{}, fmt.Errorf("creating get request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("performing get request: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		_, _ = io.Copy(ioutil.Discard, res.Body)
		return jobs.Job{}, errNoJob

	case http.StatusOK:
		var job jobs.Job
		_ = json.NewDecoder(res.Body).Decode(&job)
		return job, nil

	default:
	}

	return jobs.Job{}, fmt.Errorf("server error, recived status code %d and body", res.StatusCode)
}

func jobUpdate(ctx context.Context, host, token string, jobUpdate database.UpdateParams) (jobs.Job, error) {
	buf := bytes.NewBufferString("")
	_ = json.NewEncoder(buf).Encode(&jobUpdate)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/experiments/%s", host, jobUpdate.Id.String()), buf)
	req.Header.Set("Authorization", token)

	if err != nil {
		return jobs.Job{}, fmt.Errorf("creating get request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("performing get request: %w", err)
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		_, _ = io.Copy(ioutil.Discard, res.Body)
		return jobs.Job{}, errNoJob

	case http.StatusOK:
		var job jobs.Job
		_ = json.NewDecoder(res.Body).Decode(&job)
		return job, nil

	default:
	}

	return jobs.Job{}, fmt.Errorf("server error, recived status code %d and body", res.StatusCode)
}

func getLogs(ctx context.Context, host, token string, id uuid.UUID) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/logs/%s", host, id.String()), nil)
	if err != nil {
		return "", fmt.Errorf("creating get request: %w", err)
	}
	req.Header.Set("Authorization", token)

	res, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing get request: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		_, _ = io.Copy(ioutil.Discard, res.Body)
		return "", errNoJob

	case http.StatusOK:

		ret, _ := io.ReadAll(res.Body)
		return string(ret), nil

	default:
	}

	return "", fmt.Errorf("server error, recived status code %d and body", res.StatusCode)
}

func getLogsFollow(ctx context.Context, host, token string, id uuid.UUID, out *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/logs/%s/tail", host, id.String()), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)

	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing tail request: %w", err)
	}
	defer io.Copy(ioutil.Discard, res.Body)
	defer res.Body.Close()

	if _, err := io.Copy(out, res.Body); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		return fmt.Errorf("error reading logs: %w", err)
	}

	return nil
}
