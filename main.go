package main

import (
	"context"
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

	resources := container.Resources{
		DeviceRequests: []container.DeviceRequest{
			{
				Driver: "nvidia",
				// Count:        -1,
				DeviceIDs:    []string{"0"}, // especificar que es vol utilitzar la GPU 0
				Capabilities: [][]string{{"compute", "utility", "gpu"}},
			},
		},
	}
	hostConfig := &container.HostConfig{
		Resources: resources,
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "nvidia/cuda:11.0-base",
		Cmd:   []string{"nvidia-smi"},
	}, hostConfig, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		panic(err)
	}

	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
		panic(err)
	}
}
