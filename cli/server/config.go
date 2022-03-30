package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"
)

type HttpConfig struct {
	Listen       string        `yaml:"listen"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type QueueConfig struct {
	GPU string `yaml:"gpu"`
}

type Config struct {
	Http   HttpConfig    `yaml:"http"`
	Queues []QueueConfig `yaml:"queues"`
}

func FromFile(filename string) (*Config, error) {
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
