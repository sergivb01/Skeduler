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
	"time"

	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

var errNoJob = errors.New("no job available")

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func fetchJobs(ctx context.Context, host string, token string) (jobs.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/workers/poll", host), nil)
	req.Header.Set("User-Agent", "Skeduler-Puller/1.0")
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

func updateJob(ctx context.Context, host string, job jobs.Job, token string) error {
	buff := &bytes.Buffer{}
	if err := json.NewEncoder(buff).Encode(job); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/experiments/%s", host, job.ID), buff)
	req.Header.Set("User-Agent", "Skeduler-Puller/1.0")
	req.Header.Set("Authorization", token)
	if err != nil {
		return fmt.Errorf("creating post request: %w", err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing psot request: %w", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode == http.StatusOK {
		return nil
	}

	b, _ := ioutil.ReadAll(res.Body)

	return fmt.Errorf("server error, recived status code %d and body: %v", res.StatusCode, string(b))
}
