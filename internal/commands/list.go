package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leonnardo/orbit/internal/git"
	"github.com/leonnardo/orbit/internal/hub"
	"github.com/leonnardo/orbit/internal/state"
)

const listUsage = `usage: orbit list`

func List(args []string) error {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fmt.Println(listUsage)
			return nil
		}
		return errors.New(listUsage)
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

	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	fmt.Print(formatList(h.Config.Project, h.Config.Remote, h.Root, entries, home, cwd))
	return nil
}

// formatList renders the human-readable listing.
//
// Only worktrees strictly inside hubRoot are included; the bare entry and any
// external worktrees of the same bare are skipped. Rows are sorted by name
// (path relative to hubRoot) for deterministic output.
//
// If cwd is inside one of the listed worktrees, that row is prefixed with "*";
// all other rows get a leading space. An empty cwd (or a cwd outside every
// worktree) yields a leading space on every row.
func formatList(project, remote, hubRoot string, entries []git.WorktreeEntry, home, cwd string) string {
	hubNorm := normalizePath(hubRoot)
	hubPrefix := hubNorm + string(filepath.Separator)

	type row struct{ name, branch, path, absPath string }
	var rows []row
	maxName, maxBranch := 0, 0
	for _, e := range entries {
		if e.Bare {
			continue
		}
		eNorm := normalizePath(e.Path)
		if !strings.HasPrefix(eNorm, hubPrefix) {
			continue
		}
		rel, err := filepath.Rel(hubNorm, eNorm)
		if err != nil || rel == "" || rel == "." {
			rel = filepath.Base(eNorm)
		}
		branch := e.BranchName()
		if branch == "" || e.Detached {
			branch = "(detached)"
		}
		rows = append(rows, row{
			name:    rel,
			branch:  branch,
			path:    renderPath(eNorm, home),
			absPath: eNorm,
		})
		if len(rel) > maxName {
			maxName = len(rel)
		}
		if len(branch) > maxBranch {
			maxBranch = len(branch)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	var b strings.Builder
	if remote != "" {
		fmt.Fprintf(&b, "%s  (%s)\n", project, remote)
	} else {
		fmt.Fprintf(&b, "%s\n", project)
	}
	if len(rows) == 0 {
		b.WriteString("  (no worktrees yet — create one with `orbit new <branch>`)\n")
		return b.String()
	}
	for _, r := range rows {
		marker := " "
		if cwd != "" && isInside(cwd, r.absPath) {
			marker = "*"
		}
		fmt.Fprintf(&b, "%s %-*s  %-*s  %s\n", marker, maxName, r.name, maxBranch, r.branch, r.path)
	}
	return b.String()
}

// renderPath replaces a leading $HOME with ~ for display, otherwise returns p.
func renderPath(p, home string) string {
	if home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}
