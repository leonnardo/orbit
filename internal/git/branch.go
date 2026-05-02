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
