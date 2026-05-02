// Package git wraps git CLI invocations.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes git with the given args and returns trimmed stdout.
//
// stderr is captured and embedded in the returned error on failure.
func Run(args ...string) (string, error) {
	return runIn("", args...)
}

// RunInDir runs git with a working directory.
func RunInDir(dir string, args ...string) (string, error) {
	return runIn(dir, args...)
}

// RunGitDir runs git with --git-dir=<gitDir>.
func RunGitDir(gitDir string, args ...string) (string, error) {
	full := append([]string{"--git-dir=" + gitDir}, args...)
	return runIn("", full...)
}

func runIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// Stream runs git with stdout/stderr passed through to the parent process.
//
// Use this for long-running commands like fetch where the user benefits from
// seeing progress.
func Stream(args ...string) error {
	return streamIn("", args...)
}

// StreamGitDir is Stream with --git-dir=<gitDir>.
func StreamGitDir(gitDir string, args ...string) error {
	full := append([]string{"--git-dir=" + gitDir}, args...)
	return streamIn("", full...)
}

func streamIn(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed", strings.Join(args, " "))
	}
	return nil
}
