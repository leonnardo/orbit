// Package hub manages the per-hub .orbit.yaml config.
package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigFilename is the marker file that identifies an orbit hub.
const ConfigFilename = ".orbit.yaml"

// Config is the contents of <hub>/.orbit.yaml.
type Config struct {
	Version   int       `yaml:"version"`
	Project   string    `yaml:"project"`
	Remote    string    `yaml:"remote"`
	CreatedAt time.Time `yaml:"createdAt"`
}

// Read parses the hub config at the given file path.
func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read hub config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse hub config %s: %w", path, err)
	}
	if cfg.Version == 0 {
		return nil, fmt.Errorf("hub config %s has no version", path)
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("hub config %s has empty project", path)
	}
	return &cfg, nil
}

// Write atomically writes the hub config to the given file path.
func Write(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal hub config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create hub dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write hub config: %w", err)
	}
	return nil
}
