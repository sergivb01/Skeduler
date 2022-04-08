package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type QueueConfig struct {
	Name string   `yaml:"name"`
	GPUs []string `yaml:"gpus"`
}

type Config struct {
	Queues []QueueConfig `yaml:"queues"`
}

func configFromFile(filename string) (*Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml: %w", err)
	}

	return &c, nil
}
