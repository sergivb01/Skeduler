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
	fmt.Printf("id=%s\n", resp.ID)

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	go func(id string) {
		logs, err := cli.ContainerLogs(ctx, id, types.ContainerLogsOptions{
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

		f, err := os.OpenFile(fmt.Sprintf("./logs/%s.log", id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Printf("error closing log file: %s", err)
			}
		}()

		n, err := stdcopy.StdCopy(f, f, logs)
		if err != nil {
			panic(err)
		}
		fmt.Printf("read %d log bytes\n", n)
	}(resp.ID)

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	fmt.Printf("shutdown!\n")
}
