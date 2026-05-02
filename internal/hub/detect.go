package hub

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotInHub is returned by Detect when no hub is found in the ancestry.
var ErrNotInHub = errors.New("not inside an orbit hub")

// Hub describes a detected hub.
type Hub struct {
	Root   string  // absolute path to the hub root (the dir containing .orbit.yaml)
	Config *Config // parsed config
}

// Detect walks up from startDir looking for .orbit.yaml.
//
// If startDir is empty, the current working directory is used.
func Detect(startDir string) (*Hub, error) {
	if startDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get cwd: %w", err)
		}
		startDir = cwd
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", startDir, err)
	}

	for {
		candidate := filepath.Join(dir, ConfigFilename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			cfg, err := Read(candidate)
			if err != nil {
				return nil, err
			}
			return &Hub{Root: dir, Config: cfg}, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotInHub
		}
		dir = parent
	}
}
