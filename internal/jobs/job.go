package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gofrs/uuid"
)

type Docker struct {
	Image       string                 `json:"image" db:"docker_image"`
	Command     string                 `json:"command" db:"docker_command"`
	Environment map[string]interface{} `json:"environment" db:"docker_environment"`
}

type JobStatus string

const (
	Enqueued  JobStatus = "ENQUEUED"
	Running   JobStatus = "RUNNING"
	Finished  JobStatus = "FINISHED"
	Cancelled JobStatus = "CANCELLED"
)

type Job struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Docker      Docker      `json:"docker" db:"docker_embedded"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
	Status      JobStatus   `json:"status" db:"status"`
	Metadata    interface{} `json:"metadata" db:"metadata"`
}

const MagicEnd = "_#$#$#$<END>#$#$#$_"

var disableGpu = os.Getenv("SKEDULER_DISABLE_GPU") != ""

func NewFromFile(filename string) (*Job, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	var r Job
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

func (j *Job) Run(ctx context.Context, cli *client.Client, gpus []string, logWriter io.Writer) error {
	logr := log.New(logWriter, fmt.Sprintf("[E-%.8s] ", j.ID.String()), log.LstdFlags|log.Lmsgprefix)
	// TODO(@sergivb01): posar tots els experiments en una mateixa xarxa de docker
	// TODO(@sergivb01): no fa pull d'imatges locals???
	// la variable reader conté el progrés/log del pull de la imatge.
	// reader, err := cli.ImagePull(ctx, j.Docker.Image, types.ImagePullOptions{
	// 	// TODO(@sergivb01): pas de registre autenticació amb funció de authCredentials
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
	j.Docker.Environment["SKEDULER_GPUS"] = fmt.Sprintf("%s", gpus)

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

	if !disableGpu {
		hostConfig.Resources.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				DeviceIDs:    gpus, // especificar que es vol utilitzar la GPU 0, també podria ser "all"
				Capabilities: [][]string{{"compute", "utility"}},
			},
		}
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}
	for _, warning := range resp.Warnings {
		logr.Printf("container create warning: %v", warning)
	}
	containerID := resp.ID

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}

	containerLogs, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
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

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
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
