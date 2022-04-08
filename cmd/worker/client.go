package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

var errNoJob = errors.New("no job available")

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func fetchJobs(ctx context.Context) (jobs.Job, error) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/workers/poll", nil)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("creating get request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("performing get request: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNotFound:
		return jobs.Job{}, errNoJob

	case http.StatusOK:
		var job jobs.Job
		_ = json.NewDecoder(res.Body).Decode(&job)
		return job, nil

	default:
	}

	b, _ := ioutil.ReadAll(res.Body)
	return jobs.Job{}, fmt.Errorf("server error, recived status code %d and body: %v", res.StatusCode, string(b))
}

func updateJob(ctx context.Context, job jobs.Job) error {
	buff := &bytes.Buffer{}
	_ = json.NewEncoder(buff).Encode(job)

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("http://localhost:8080/%s", job.ID), buff)
	if err != nil {
		return fmt.Errorf("creating post request: %w", err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing psot request: %w", err)
	}

	if res.StatusCode == http.StatusOK {
		return nil
	}

	b, _ := ioutil.ReadAll(res.Body)

	return fmt.Errorf("server error, recived status code %d and body: %v", res.StatusCode, string(b))
}
