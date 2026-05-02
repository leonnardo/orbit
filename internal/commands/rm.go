package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leonnardo/orbit/internal/git"
	"github.com/leonnardo/orbit/internal/hub"
	"github.com/leonnardo/orbit/internal/state"
)

const rmUsage = `usage: orbit rm <path-or-name> [--delete-branch]`

func Rm(args []string) error {
	target, deleteBranch, err := parseRmArgs(args)
	if err != nil {
		return err
	}

	h, err := hub.Detect("")
	if err != nil {
		if errors.Is(err, hub.ErrNotInHub) {
			return errors.New("not inside an orbit hub\n  run `cd <project>` first, where <project> contains .orbit.yaml")
		}
		return err
	}

	barePath, err := state.BarePath(h.Config.Project)
	if err != nil {
		return err
	}
	if _, err := os.Stat(barePath); err != nil {
		return fmt.Errorf("bare repo missing for project %q: %s", h.Config.Project, barePath)
	}

	entries, err := git.ListWorktrees(barePath)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	wt, err := resolveRmTarget(target, cwd, h.Root, entries)
	if err != nil {
		return err
	}

	branch := wt.BranchName()
	if deleteBranch && branch == "" {
		return fmt.Errorf("--delete-branch requested but worktree %s has no branch (detached HEAD)", wt.Path)
	}

	if deleteBranch {
		if head := git.HeadBranch(barePath); head != "" && head == branch {
			return fmt.Errorf("cannot delete branch %q: it is the bare repo's HEAD\n  set HEAD elsewhere first:\n    git --git-dir=%s symbolic-ref HEAD refs/heads/<other-branch>",
				branch, barePath)
		}
	}

	if _, err := os.Stat(wt.Path); err != nil {
		return fmt.Errorf("worktree path missing on disk: %s\n  prune stale entries with:\n    git --git-dir=%s worktree prune",
			wt.Path, barePath)
	}

	selfRemoval := isInside(cwd, wt.Path)

	fmt.Fprintf(os.Stderr, "orbit: removing worktree %s\n", wt.Path)
	if err := git.RemoveWorktree(barePath, wt.Path); err != nil {
		return err
	}

	if deleteBranch {
		fmt.Fprintf(os.Stderr, "orbit: deleting branch %s\n", branch)
		if err := git.DeleteBranch(barePath, branch); err != nil {
			return fmt.Errorf("worktree removed but branch delete failed: %w\n  delete it manually with:\n    git --git-dir=%s branch -d %s",
				err, barePath, branch)
		}
	}

	fmt.Fprintln(os.Stderr, "orbit: done")
	if selfRemoval {
		fmt.Fprintln(os.Stderr, "orbit: warning: your shell is inside the removed directory; cd elsewhere")
	}
	return nil
}

func parseRmArgs(args []string) (target string, deleteBranch bool, err error) {
	for _, a := range args {
		switch {
		case a == "--delete-branch":
			deleteBranch = true
		case a == "-h" || a == "--help":
			return "", false, errors.New(rmUsage)
		case strings.HasPrefix(a, "-"):
			return "", false, fmt.Errorf("unknown flag %q\n%s", a, rmUsage)
		default:
			if target != "" {
				return "", false, errors.New(rmUsage)
			}
			target = a
		}
	}
	if target == "" {
		return "", false, errors.New(rmUsage)
	}
	return target, deleteBranch, nil
}

// resolveRmTarget locates a worktree entry from a user-supplied target.
//
// To avoid destructive ambiguity, the target is interpreted as either:
//   - an explicit path (absolute, starts with "." or "..", or contains a path
//     separator) — joined with cwd if relative, then matched against entries; or
//   - a worktree basename inside the hub root — matched as <hubRoot>/<target>.
//
// Only worktrees strictly inside the hub root are considered. The bare entry is
// always skipped.
func resolveRmTarget(target, cwd, hubRoot string, entries []git.WorktreeEntry) (*git.WorktreeEntry, error) {
	var candidate string
	if isExplicitPath(target) {
		if filepath.IsAbs(target) {
			candidate = filepath.Clean(target)
		} else {
			candidate = filepath.Clean(filepath.Join(cwd, target))
		}
	} else {
		candidate = filepath.Clean(filepath.Join(hubRoot, target))
	}

	targetNorm := normalizePath(candidate)
	hubNorm := normalizePath(hubRoot)
	hubPrefix := hubNorm + string(filepath.Separator)

	for i := range entries {
		e := &entries[i]
		if e.Bare {
			continue
		}
		eNorm := normalizePath(e.Path)
		if !strings.HasPrefix(eNorm, hubPrefix) {
			continue
		}
		if eNorm == targetNorm {
			return e, nil
		}
	}

	return nil, fmt.Errorf("no orbit worktree matches %q\n  resolved to: %s\n  hub root:    %s", target, candidate, hubRoot)
}

func isExplicitPath(target string) bool {
	if filepath.IsAbs(target) {
		return true
	}
	if target == "." || target == ".." || strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") {
		return true
	}
	return strings.ContainsAny(target, `/\`)
}

func normalizePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

func isInside(child, parent string) bool {
	c := normalizePath(child)
	p := normalizePath(parent)
	if c == p {
		return true
	}
	return strings.HasPrefix(c, p+string(filepath.Separator))
}
