package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type worker struct {
	id    int
	cli   *client.Client
	reqs  <-chan jobs.Job
	quit  chan<- struct{}
	gpus  []string
	token string
	host  string
}

func (w *worker) start() {
	for t := range w.reqs {
		if err := w.run(t); err != nil {
			log.Printf("error running task: %s", err)
			t.Status = jobs.Cancelled
		} else {
			t.Status = jobs.Finished
		}

		if err := updateJob(context.TODO(), w.host, t, w.token); err != nil {
			log.Printf("failed to update job %+v status: %v\n", t.ID, err)
		}
	}
	w.quit <- struct{}{}
}

func puller(tasks chan<- jobs.Job, closing <-chan struct{}, host string, token string) {
	t := time.NewTicker(time.Second * 3)
	defer t.Stop()

	for {
		select {
		case <-closing:
			return
		case <-t.C:
			// all workers are being used
			if len(tasks) == cap(tasks) {
				continue
			}

			job, err := fetchJobs(context.TODO(), host, token)
			if err != nil {
				// no job available
				if errors.Is(err, errNoJob) {
					continue
				}
				log.Printf("error pulling: %v\n", err)
				continue
			}

			tasks <- job

			break
		}
	}
}

func (w *worker) run(j jobs.Job) error {
	u, err := url.Parse(w.host)
	if err != nil {
		return err
	}
	u.Path = fmt.Sprintf("/logs/%s/upload", j.ID)
	u.Scheme = "ws"

	lr := &websocketWriter{
		mu:    &sync.Mutex{},
		buff:  &bytes.Buffer{},
		uri:   u.String(),
		token: w.token,
	}

	if err := lr.connect(); err != nil {
		return fmt.Errorf("error connecting ws: %w", err)
	}
	defer lr.Close()

	// logging to stderr as well as the custom log io.Writer
	logWriter := io.MultiWriter(os.Stderr, lr)
	logr := log.New(logWriter, fmt.Sprintf("[E-%.8s] ", j.ID.String()), log.LstdFlags|log.Lmsgprefix)

	t := time.NewTicker(time.Millisecond * 500)
	defer t.Stop()
	go func() {
		for range t.C {
			if err := lr.Flush(); err != nil {
				log.Printf("error flushing logs: %v", err)
			}
		}
	}()

	defer func() {
		_, _ = logWriter.Write([]byte(jobs.MagicEnd))
		_, _ = logWriter.Write([]byte{'\n'})
	}()

	logr.Printf("[%d] worker running task %+v at %s\n", w.id, j, time.Now())

	ctx := context.TODO()
	// TODO(@sergivb01): no fa pull d'imatges locals???
	// la variable reader conté el progrés/log del pull de la imatge.
	// reader, err := cli.ImagePull(ctx, j.Docker.Image, types.ImagePullOptions{
	// 	// pas de registre autenticació amb funció de authCredentials
	// 	// RegistryAuth: "",
	// })
	// if err != nil {
	// 	return fmt.Errorf("pulling docker image: %w", err)
	// }

	// if _, err := io.Copy(logWriter, reader); err != nil {
	// 	logr.Printf("error copying pull output to log for %v: %v\n", j.ID, err)
	// }

	logr.Printf("starting task at %s", time.Now())
	if j.Docker.Environment == nil {
		j.Docker.Environment = make(map[string]interface{})
	}
	// establir variables d'entorn que també volem guardar a la base de dades
	j.Docker.Environment["SKEDULER_GPUS"] = fmt.Sprintf("%s", w.gpus)

	envNew := make(map[string]interface{})
	for k, v := range j.Docker.Environment {
		envNew[k] = v
	}

	// pas de variables entorn com ID de la tasca, prioritat, GPUs, ...
	envNew["SKEDULER_ID"] = j.ID
	envNew["SKEDULER_NAME"] = j.Name
	envNew["SKEDULER_DESCRIPTION"] = j.Description
	envNew["SKEDULER_DOCKER_IMAGE"] = j.Docker.Image
	envNew["SKEDULER_DOCKER_COMMAND"] = j.Docker.Command

	var env []string
	for k, v := range j.Docker.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	cmd := strings.Split(j.Docker.Command, " ")
	containerConfig := &container.Config{
		Image:    j.Docker.Image,
		Cmd:      cmd,
		Hostname: fmt.Sprintf("exp_%.8s", j.ID.String()),
		Env:      env,
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Resources:  container.Resources{
			// CPUCount: 2,
			// Memory:   1024 * 1024 * 256, // 256mb
		},
	}

	if len(w.gpus) != 0 {
		hostConfig.Resources.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				DeviceIDs:    w.gpus, // especificar que es vol utilitzar la GPU 0, també podria ser "all"
				Capabilities: [][]string{{"compute", "utility"}},
			},
		}
	}

	resp, err := w.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		logr.Printf("error creating container: %v", err)
		return fmt.Errorf("creating container: %w", err)
	}
	for _, warning := range resp.Warnings {
		logr.Printf("container create warning: %v", warning)
	}
	containerID := resp.ID

	if err := w.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		logr.Printf("error starting container: %v", err)
		return fmt.Errorf("starting container: %w", err)
	}

	containerLogs, err := w.cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
		Tail:       "all",
		Details:    true,
	})
	if err != nil {
		_ = w.cli.ContainerStop(ctx, containerID, nil)
		logr.Printf("error getting container logs, stopping container: %v", err)
		return fmt.Errorf("getting logs: %w", err)
	}
	defer containerLogs.Close()

	doneLogs := make(chan struct{}, 1)
	go func() {
		_, err := stdcopy.StdCopy(logWriter, logWriter, containerLogs)
		if err != nil {
			logr.Printf("error copying logs to file: %v\n", err)
			return
		}
		doneLogs <- struct{}{}
	}()

	statusCh, errCh := w.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logr.Printf("container %s received error: %v", j.ID, containerID, err)
		}
	case s := <-statusCh:
		logr.Printf("container %s stopped with status code = %v and error = %v\n", containerID, s.StatusCode, s.Error)
		break
	}

	<-doneLogs

	return nil
}

func authCredentials(username, password string) (string, error) {
	authConfig := types.AuthConfig{
		Username: username,
		Password: password,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("marshaling authconfig to json: %w", err)
	}

	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}
