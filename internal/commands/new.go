package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/leonnardo/orbit/internal/git"
	"github.com/leonnardo/orbit/internal/hub"
	"github.com/leonnardo/orbit/internal/slug"
	"github.com/leonnardo/orbit/internal/state"
)

const newUsage = `usage: orbit new <branch> [path]`

func New(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New(newUsage)
	}
	branch := args[0]
	var userPath string
	if len(args) == 2 {
		userPath = args[1]
	}

	h, err := hub.Detect("")
	if err != nil {
		if errors.Is(err, hub.ErrNotInHub) {
			return errors.New("not inside an orbit hub\nrun `orbit clone <repo-url> [project]` first, then `cd <project>`")
		}
		return err
	}

	barePath, err := state.BarePath(h.Config.Project)
	if err != nil {
		return err
	}
	if _, err := os.Stat(barePath); err != nil {
		return fmt.Errorf("bare repo missing for project %q: %s\n(automatic recovery is out of scope for the MVP — re-clone the project)", h.Config.Project, barePath)
	}

	fmt.Fprintln(os.Stderr, "orbit: fetching")
	if err := git.StreamGitDir(barePath, "fetch", "--prune"); err != nil {
		return err
	}

	action, err := git.ResolveBranch(barePath, branch)
	if err != nil {
		return err
	}

	if action != git.CreateNew {
		if existing, checkedOut, err := git.CheckedOutAt(barePath, branch); err != nil {
			return err
		} else if checkedOut {
			return fmt.Errorf("branch is already checked out: %s\n  existing worktree: %s", branch, existing)
		}
	}

	switch action {
	case git.UseLocal:
		fmt.Fprintf(os.Stderr, "orbit: opening existing local branch %s\n", branch)
	case git.CreateTracking:
		fmt.Fprintf(os.Stderr, "orbit: tracking origin/%s as new local branch\n", branch)
		if err := git.CreateTrackingBranch(barePath, branch); err != nil {
			return err
		}
	case git.CreateNew:
		fmt.Fprintf(os.Stderr, "orbit: creating new branch %s from origin/HEAD\n", branch)
		if err := git.CreateNewBranch(barePath, branch); err != nil {
			return err
		}
	}

	wtPath, err := hub.ResolveWorktreePath(h.Root, slug.Branch(branch), userPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(wtPath); err == nil {
		return fmt.Errorf("worktree path already exists: %s", wtPath)
	}

	fmt.Fprintf(os.Stderr, "orbit: creating worktree at %s\n", wtPath)
	if err := git.AddWorktree(barePath, wtPath, branch); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "orbit: worktree ready: %s\n", wtPath)
	return nil
}
