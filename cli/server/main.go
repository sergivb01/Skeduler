package main

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Resources: container.Resources{
			DeviceRequests: []container.DeviceRequest{
				{
					Driver: "nvidia",
					// Count:        -1,
					DeviceIDs:    []string{"0"}, // especificar que es vol utilitzar la GPU 0
					Capabilities: [][]string{{"compute", "utility"}},
				},
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "nvidia/cuda:11.0-base",
		Cmd:   []string{"nvidia-smi"},
	}, hostConfig, nil, nil, "")
	if err != nil {
		panic(err)
	}
	containerID := resp.ID
	fmt.Printf("id=%s\n", containerID)

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		panic(err)
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
		panic(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("./logs/%s.log", containerID), os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	go func(id string) {
		n, err := stdcopy.StdCopy(f, f, logs)
		if err != nil {
			panic(err)
		}
		fmt.Printf("read %d log bytes\n", n)
	}(containerID)

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case s := <-statusCh:
		fmt.Printf("container %s stopped with status code = %d and error = %q\n", containerID, s.StatusCode, s.Error)
		break
	}

	fmt.Printf("shutdown!\n")
}
