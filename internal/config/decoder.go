package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var fileDecoders = map[string]func(data []byte, v interface{}) error{
	"json": json.Unmarshal,
	"yml":  yaml.Unmarshal,
	"yaml": yaml.Unmarshal,
}

func DecodeFromFile[T any](filename string) (*T, error) {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := filepath.Ext(filename)
	dec, ok := fileDecoders[strings.ToLower(ext)[1:]]
	if !ok {
		return nil, fmt.Errorf("cannot get decoder for file %s with extension %q", filename, ext)
	}

	var c T
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := dec(b, &c); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}

	return &c, nil
}
