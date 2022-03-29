package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Docker struct {
	Image       string                 `json:"image"`
	Command     string                 `json:"command"`
	Environment map[string]interface{} `json:"environment"`
}

type JobRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Request     struct {
		Docker Docker `json:"docker"`
	} `json:"request"`
}

func NewFromFile(filename string) (*JobRequest, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var r JobRequest
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	return &r, err
}

func (r *JobRequest) Run(ctx context.Context, cli *client.Client, gpus []string) error {
	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Resources: container.Resources{
			// CPUCount: 2,
			// Memory:   1024 * 1024 * 256, // 256mb
			DeviceRequests: []container.DeviceRequest{
				{
					Driver: "nvidia",
					// Count:        -1,
					DeviceIDs:    gpus, // especificar que es vol utilitzar la GPU 0, també podria ser "all"
					Capabilities: [][]string{{"compute", "utility"}},
				},
			},
		},
	}

	reader, err := cli.ImagePull(ctx, r.Request.Docker.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pulling docker image: %w", err)
	}
	_, _ = io.Copy(ioutil.Discard, reader)

	// TODO(@sergivb01): pas de variables entorn com ID de la tasca, prioritat, GPUs, ...
	r.Request.Docker.Environment["SKEDULER_ID"] = "test12345"

	var env []string
	for k, v := range r.Request.Docker.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	cmd := strings.Split(r.Request.Docker.Command, " ")
	containerConfig := &container.Config{
		Image: r.Request.Docker.Image,
		Cmd:   cmd,
		// Hostname: "hostname",
		// Domainname: "",
		Env: env,
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}
	containerID := resp.ID

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}

	logs, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
		Tail:       "all",
		Details:    true,
	})
	if err != nil {
		return fmt.Errorf("getting logs: %w", err)
	}

	f, err := os.OpenFile(fmt.Sprintf("./logs/%s.log", containerID), os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("creating logs file: %w", err)
	}
	defer f.Close()

	go func() {
		w := bufio.NewWriter(f)
		defer w.Flush()
		n, err := stdcopy.StdCopy(w, w, logs)
		if err != nil {
			log.Printf("copying logs to file: %s\n", err)
			return
		}
		log.Printf("read %d log bytes\n", n)
	}()

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case s := <-statusCh:
		fmt.Printf("container %.7s stopped with status code = %d and error = %q\n", containerID, s.StatusCode, s.Error)
		break
	}

	return nil
}
