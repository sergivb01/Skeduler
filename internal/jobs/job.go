package jobs

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

const MAGIC_END = "_#$#$#$<END>#$#$#$_"

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

func (j *Job) Run(ctx context.Context, cli *client.Client, gpus []string) error {
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

	logFile, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", j.ID), os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("creating logs file: %w", err)
	}
	logWriter := bufio.NewWriter(logFile)

	t := time.NewTicker(time.Second)
	defer t.Stop()
	go func() {
		for range t.C {
			_ = logWriter.Flush()
		}
	}()

	defer func() {
		_, _ = logWriter.Write([]byte(MAGIC_END))
		_, _ = logWriter.Write([]byte{'\n'})
		if err := logWriter.Flush(); err != nil {
			log.Printf("error flushing logs for %v: %s\n", j.ID, err)
		}
		if err := logFile.Close(); err != nil {
			log.Printf("error closing log file for %v: %s\n", j.ID, err)
		}
	}()

	// if _, err := io.Copy(logWriter, reader); err != nil {
	// 	log.Printf("error copying pull output to log for %v: %v\n", j.ID, err)
	// }

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
		Image: j.Docker.Image,
		Cmd:   cmd,
		// TODO(@sergivb01): canviar el hostname i el domini per alguna cosa més significativa
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
	for _, warning := range resp.Warnings {
		_, _ = fmt.Fprintf(logWriter, "[CONTAINER CREATE WARNING] %v\n", warning)
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
		n, err := stdcopy.StdCopy(logWriter, logWriter, containerLogs)
		if err != nil {
			log.Printf("error copying logs to file: %s\n", err)
			return
		}
		log.Printf("read %d log bytes from %s\n", n, containerID)
		doneLogs <- struct{}{}
	}()

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("job %v with container %s recived error: %v", j.ID, containerID, err)
		}
	case s := <-statusCh:
		log.Printf("job %v with container %s stopped with status code = %d and error = %v\n", j.ID, containerID, s.StatusCode, s.Error)
		break
	}

	<-doneLogs

	return nil
}
