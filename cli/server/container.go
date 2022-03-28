package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Docker struct {
	Image       string            `json:"image"`
	Command     string            `json:"command"`
	Environment map[string]string `json:"environment"`
}

type JobRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Request     struct {
		Docker Docker `json:"docker"`
	} `json:"request"`
}

func NewFromFile(filename string) (*JobRequest, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var r JobRequest
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %w", err)
	}
	r.ID = ""

	return &r, err
}
