package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/leonnardo/orbit/internal/git"
	"github.com/leonnardo/orbit/internal/hub"
	"github.com/leonnardo/orbit/internal/project"
	"github.com/leonnardo/orbit/internal/state"
)

const cloneUsage = `usage: orbit clone <repo-url-or-path> [project]`

func Clone(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New(cloneUsage)
	}
	remote := args[0]

	var name string
	var err error
	if len(args) == 2 {
		name = args[1]
		if err := project.Validate(name); err != nil {
			return err
		}
	} else {
		name, err = project.DeriveFromURL(remote)
		if err != nil {
			return err
		}
	}

	barePath, err := state.BarePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(barePath); err == nil {
		return fmt.Errorf("project already exists in state: %s\nuse a different local name: orbit clone <repo-url> <project>", barePath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}
	hubPath := filepath.Join(cwd, name)
	if _, err := os.Stat(hubPath); err == nil {
		return fmt.Errorf("hub directory already exists: %s\nuse a different local name: orbit clone <repo-url> <project>", hubPath)
	}

	if _, err := state.EnsureReposDir(); err != nil {
		return err
	}

	bareCreated := false
	hubCreated := false
	defer func() {
		// Best-effort cleanup on failure.
		if err == nil {
			return
		}
		if hubCreated {
			os.RemoveAll(hubPath)
		}
		if bareCreated {
			os.RemoveAll(barePath)
		}
	}()

	fmt.Fprintf(os.Stderr, "orbit: creating bare repo at %s\n", barePath)
	if _, gerr := git.Run("init", "--bare", barePath); gerr != nil {
		err = gerr
		return err
	}
	bareCreated = true

	if _, gerr := git.RunGitDir(barePath, "remote", "add", "origin", remote); gerr != nil {
		err = gerr
		return err
	}

	fmt.Fprintf(os.Stderr, "orbit: fetching from %s\n", remote)
	if ferr := git.StreamGitDir(barePath, "fetch", "origin",
		"+refs/heads/*:refs/remotes/origin/*",
		"+refs/tags/*:refs/tags/*",
		"--prune"); ferr != nil {
		err = ferr
		return err
	}

	// remote set-head can fail on empty remotes; warn but don't abort.
	if _, gerr := git.RunGitDir(barePath, "remote", "set-head", "origin", "-a"); gerr != nil {
		fmt.Fprintf(os.Stderr, "orbit: warning: could not set origin/HEAD: %v\n", gerr)
	}

	if mkErr := os.MkdirAll(hubPath, 0o755); mkErr != nil {
		err = fmt.Errorf("create hub: %w", mkErr)
		return err
	}
	hubCreated = true

	cfg := &hub.Config{
		Version:   1,
		Project:   name,
		Remote:    remote,
		CreatedAt: time.Now().UTC(),
	}
	cfgPath := filepath.Join(hubPath, hub.ConfigFilename)
	if werr := hub.Write(cfgPath, cfg); werr != nil {
		err = werr
		return err
	}

	fmt.Fprintf(os.Stderr, "orbit: hub ready at %s\n", hubPath)
	return nil
}
