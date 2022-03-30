package main

import (
	"bufio"
	"context"
	"encoding/base64"
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
	Docker      Docker `json:"docker"`
}

func NewFromFile(filename string) (*JobRequest, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	var r JobRequest
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("unmarshaling json: %w", err)
	}

	return &r, nil
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

func (r *JobRequest) Run(ctx context.Context, cli *client.Client, gpus []string) error {
	reader, err := cli.ImagePull(ctx, r.Docker.Image, types.ImagePullOptions{
		// TODO(@sergivb01): pas de registre autenticació amb funció de authCredentials
		RegistryAuth: "",
	})
	if err != nil {
		return fmt.Errorf("pulling docker image: %w", err)
	}
	_, _ = io.Copy(ioutil.Discard, reader)

	// TODO(@sergivb01): pas de variables entorn com ID de la tasca, prioritat, GPUs, ...
	r.Docker.Environment["SKEDULER_ID"] = "test12345"

	var env []string
	for k, v := range r.Docker.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	cmd := strings.Split(r.Docker.Command, " ")
	containerConfig := &container.Config{
		Image: r.Docker.Image,
		Cmd:   cmd,
		// Hostname: "hostname",
		// Domainname: "",
		Env: env,
	}

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
		log.Printf("read %d log bytes from %.7s\n", n, containerID)
	}()

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("error reading logs from %.7s: %v", containerID, err)
		}
	case s := <-statusCh:
		log.Printf("container %.7s stopped with status code = %d and error = %v\n", containerID, s.StatusCode, s.Error)
		break
	}

	return nil
}
