package git

import (
	"bufio"
	"fmt"
	"strings"
)

// BranchAction describes how `orbit new` should handle a branch.
type BranchAction int

const (
	// UseLocal: refs/heads/<branch> already exists.
	UseLocal BranchAction = iota
	// CreateTracking: only refs/remotes/origin/<branch> exists.
	CreateTracking
	// CreateNew: branch doesn't exist anywhere.
	CreateNew
)

func (a BranchAction) String() string {
	switch a {
	case UseLocal:
		return "use-local"
	case CreateTracking:
		return "create-tracking"
	case CreateNew:
		return "create-new"
	}
	return "unknown"
}

// ResolveBranch decides what to do with the given branch name in the bare repo.
func ResolveBranch(gitDir, branch string) (BranchAction, error) {
	if hasRef(gitDir, "refs/heads/"+branch) {
		return UseLocal, nil
	}
	if hasRef(gitDir, "refs/remotes/origin/"+branch) {
		return CreateTracking, nil
	}
	return CreateNew, nil
}

func hasRef(gitDir, ref string) bool {
	_, err := RunGitDir(gitDir, "show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// CreateTrackingBranch creates a local branch tracking origin/<branch>.
func CreateTrackingBranch(gitDir, branch string) error {
	_, err := RunGitDir(gitDir, "branch", "--track", branch, "origin/"+branch)
	return err
}

// CreateNewBranch creates a new local branch starting from origin/HEAD.
func CreateNewBranch(gitDir, branch string) error {
	_, err := RunGitDir(gitDir, "branch", branch, "origin/HEAD")
	return err
}

// WorktreeEntry is one entry from `git worktree list --porcelain`.
type WorktreeEntry struct {
	Path     string // path as reported by git (may be a symlink)
	HeadSha  string
	Branch   string // full ref e.g. "refs/heads/main"; empty for bare/detached
	Bare     bool
	Detached bool
	Locked   bool
}

// BranchName returns the short branch name (refs/heads/X → X). Empty for bare/detached.
func (e WorktreeEntry) BranchName() string {
	const p = "refs/heads/"
	if strings.HasPrefix(e.Branch, p) {
		return strings.TrimPrefix(e.Branch, p)
	}
	return ""
}

// ShortHead returns the abbreviated HEAD SHA for display.
func (e WorktreeEntry) ShortHead() string {
	if len(e.HeadSha) <= 7 {
		return e.HeadSha
	}
	return e.HeadSha[:7]
}

// ListWorktrees returns all worktrees registered with the bare repo at gitDir.
func ListWorktrees(gitDir string) ([]WorktreeEntry, error) {
	out, err := RunGitDir(gitDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out)
}

func parseWorktreeList(porcelain string) ([]WorktreeEntry, error) {
	var entries []WorktreeEntry
	var cur WorktreeEntry
	flush := func() {
		if cur.Path != "" {
			entries = append(entries, cur)
			cur = WorktreeEntry{}
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(porcelain))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			cur.HeadSha = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(line, "branch ")
		case line == "bare":
			cur.Bare = true
		case line == "detached":
			cur.Detached = true
		case strings.HasPrefix(line, "locked"):
			cur.Locked = true
		case line == "":
			flush()
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan worktree list: %w", err)
	}
	return entries, nil
}

// CheckedOutAt returns the worktree path where branch is currently checked out, if any.
func CheckedOutAt(gitDir, branch string) (string, bool, error) {
	entries, err := ListWorktrees(gitDir)
	if err != nil {
		return "", false, err
	}
	for _, e := range entries {
		if e.BranchName() == branch {
			return e.Path, true, nil
		}
	}
	return "", false, nil
}

// AddWorktree runs `git worktree add <path> <branch>` against the bare repo.
func AddWorktree(gitDir, path, branch string) error {
	_, err := RunGitDir(gitDir, "worktree", "add", path, branch)
	return err
}

// RemoveWorktree runs `git worktree remove <path>` against the bare repo.
// Without --force, git refuses to remove dirty worktrees (which is the MVP behavior).
func RemoveWorktree(gitDir, path string) error {
	_, err := RunGitDir(gitDir, "worktree", "remove", path)
	return err
}

// DeleteBranch runs `git branch -d <branch>` against the bare repo.
// Without --force, git refuses to delete unmerged branches.
func DeleteBranch(gitDir, branch string) error {
	_, err := RunGitDir(gitDir, "branch", "-d", branch)
	return err
}

// HeadBranch returns the short branch name that the bare repo's HEAD points to.
// Returns "" if HEAD is detached or unset.
func HeadBranch(gitDir string) string {
	out, err := RunGitDir(gitDir, "symbolic-ref", "--quiet", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(out), "refs/heads/")
}

// HeadMessage returns the subject of the given commit.
func HeadMessage(gitDir, sha string) (string, error) {
	if strings.TrimSpace(sha) == "" {
		return "", nil
	}
	return RunGitDir(gitDir, "log", "-1", "--format=%s", sha)
}

// WorktreeStatus summarizes the working tree state using porcelain v1 output.
type WorktreeStatus struct {
	Changed   bool
	Untracked bool
	Conflict  bool
	Ahead     bool
	Behind    bool
}

// Status returns a compact summary of the worktree's local changes.
func Status(path string) (WorktreeStatus, error) {
	out, err := RunInDir(path, "status", "--porcelain")
	if err != nil {
		return WorktreeStatus{}, err
	}
	st := ParseStatus(out)
	ahead, behind, err := upstreamCounts(path)
	if err == nil {
		st.Ahead = ahead > 0
		st.Behind = behind > 0
	}
	return st, nil
}

// ParseStatus summarizes `git status --porcelain` output.
func ParseStatus(porcelain string) WorktreeStatus {
	var st WorktreeStatus
	for _, line := range strings.Split(porcelain, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			st.Untracked = true
			continue
		}
		if len(line) >= 2 && (isUnmergedStatus(line[0:2])) {
			st.Conflict = true
			continue
		}
		st.Changed = true
	}
	return st
}

func isUnmergedStatus(s string) bool {
	switch s {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	default:
		return false
	}
}

func upstreamCounts(path string) (ahead, behind int, err error) {
	out, err := RunInDir(path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return 0, 0, err
	}
	if _, err := fmt.Sscanf(out, "%d\t%d", &ahead, &behind); err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}
