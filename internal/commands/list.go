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

	entries, err := listRows(barePath, h.Root)
	if err != nil {
		return err
	}

	cwd, _ := os.Getwd()
	fmt.Print(formatList(h.Root, entries, cwd))
	return nil
}

type listRow struct {
	current                                        bool
	branch, status, path, commit, message, absPath string
}

func listRows(barePath, hubRoot string) ([]listRow, error) {
	entries, err := git.ListWorktrees(barePath)
	if err != nil {
		return nil, err
	}

	hubNorm := normalizePath(hubRoot)
	hubPrefix := hubNorm + string(filepath.Separator)

	var rows []listRow
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
		status, err := worktreeSignals(e, eNorm)
		if err != nil {
			return nil, err
		}
		message, err := git.HeadMessage(barePath, e.HeadSha)
		if err != nil {
			return nil, err
		}
		rows = append(rows, listRow{
			branch:  branch,
			status:  status,
			path:    rel,
			commit:  e.ShortHead(),
			message: message,
			absPath: eNorm,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].path < rows[j].path })
	return rows, nil
}

func worktreeSignals(e git.WorktreeEntry, path string) (string, error) {
	st, err := git.Status(path)
	if err != nil {
		return "", err
	}
	return formatSignals(e, st), nil
}

func formatSignals(e git.WorktreeEntry, st git.WorktreeStatus) string {
	var signals strings.Builder
	if e.Locked {
		signals.WriteString("🔒")
	}
	if e.Detached {
		signals.WriteString("◆")
	}
	if st.Conflict {
		signals.WriteString("!")
	}
	if st.Changed {
		signals.WriteString("●")
	}
	if st.Untracked {
		signals.WriteString("?")
	}
	if st.Ahead {
		signals.WriteString("↑")
	}
	if st.Behind {
		signals.WriteString("↓")
	}
	if signals.Len() == 0 {
		return "✓"
	}
	return signals.String()
}

// formatList renders the human-readable listing.
//
// Rows are sorted before rendering by listRows. If cwd is inside one of the
// listed worktrees, that row is marked with "*".
func formatList(hubRoot string, rows []listRow, cwd string) string {
	const (
		hdrBranch = "BRANCH"
		hdrStatus = "STATUS"
		hdrPath   = "PATH"
		hdrCommit = "COMMIT"
		hdrMsg    = "MESSAGE"
	)
	maxBranch, maxStatus, maxPath, maxCommit := displayWidth(hdrBranch), displayWidth(hdrStatus), displayWidth(hdrPath), displayWidth(hdrCommit)
	for _, r := range rows {
		if displayWidth(r.branch) > maxBranch {
			maxBranch = displayWidth(r.branch)
		}
		if displayWidth(r.status) > maxStatus {
			maxStatus = displayWidth(r.status)
		}
		if displayWidth(r.path) > maxPath {
			maxPath = displayWidth(r.path)
		}
		if displayWidth(r.commit) > maxCommit {
			maxCommit = displayWidth(r.commit)
		}
	}

	currentPath := ""
	cwdNorm := normalizePath(cwd)
	hubNorm := normalizePath(hubRoot)
	if cwdNorm != "" && cwdNorm != hubNorm {
		for i, r := range rows {
			if isInside(cwdNorm, r.absPath) {
				currentPath = r.path
				rows[i].current = true
				break
			}
		}
	}
	markerWidth := 0
	if currentPath != "" {
		markerWidth = 1
	}

	var b strings.Builder
	if len(rows) == 0 {
		b.WriteString("(no worktrees yet — create one with `orbit new <branch>`)\n")
		return b.String()
	}
	if markerWidth > 0 {
		b.WriteString("  ")
	}
	fmt.Fprintf(&b, "%s  %s  %s  %s  %s\n", padRight(hdrBranch, maxBranch), padRight(hdrStatus, maxStatus), padRight(hdrPath, maxPath), padRight(hdrCommit, maxCommit), hdrMsg)
	for _, r := range rows {
		if markerWidth > 0 {
			marker := " "
			if r.current {
				marker = "*"
			}
			fmt.Fprintf(&b, "%s ", marker)
		}
		fmt.Fprintf(&b, "%s  %s  %s  %s  %s\n", padRight(r.branch, maxBranch), padRight(r.status, maxStatus), padRight(r.path, maxPath), padRight(r.commit, maxCommit), r.message)
	}
	return b.String()
}

func padRight(s string, width int) string {
	padding := width - displayWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		switch r {
		case '🔒':
			width += 2
		default:
			width++
		}
	}
	return width
}
