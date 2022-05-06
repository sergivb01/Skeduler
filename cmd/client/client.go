package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var errNoJob = errors.New("no job available")

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func getJobs(ctx context.Context, host string, token string) (jobs.Job, error) {
	// TODO: Falta implementar al servidor

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/experiments", host), nil)
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

func newJob(ctx context.Context, host string, token string, jobRequest database.InsertParams) (jobs.Job, error) {
	buf := bytes.NewBufferString("")
	_ = json.NewEncoder(buf).Encode(&jobRequest)

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

func getJob(ctx context.Context, host string, token string, id uuid.UUID) (jobs.Job, error) {
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

func jobUpdate(ctx context.Context, host string, token string, jobUpdate database.UpdateParams) (jobs.Job, error) {
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

func getLogs(ctx context.Context, host string, token string, id uuid.UUID) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/logs/%s", host, id.String()), nil)
	req.Header.Set("Authorization", token)

	if err != nil {
		return "", fmt.Errorf("creating get request: %w", err)
	}

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
