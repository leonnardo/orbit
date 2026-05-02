// Package state knows where orbit's global state lives on disk.
package state

import (
	"fmt"
	"os"
	"path/filepath"
)

// StateDir returns the global orbit state directory.
//
// Honors XDG_STATE_HOME if set, falling back to ~/.local/state/orbit.
func StateDir() (string, error) {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "orbit"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "state", "orbit"), nil
}

// ReposDir returns <state>/repos.
func ReposDir() (string, error) {
	s, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(s, "repos"), nil
}

// BarePath returns the bare repo path for a project.
func BarePath(project string) (string, error) {
	r, err := ReposDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, project), nil
}

// EnsureReposDir creates <state>/repos if it doesn't exist.
func EnsureReposDir() (string, error) {
	r, err := ReposDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(r, 0o755); err != nil {
		return "", fmt.Errorf("create repos dir: %w", err)
	}
	return r, nil
}
