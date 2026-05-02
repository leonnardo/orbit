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

// CheckedOutAt returns the worktree path where branch is currently checked out, if any.
func CheckedOutAt(gitDir, branch string) (string, bool, error) {
	out, err := RunGitDir(gitDir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", false, err
	}
	scanner := bufio.NewScanner(strings.NewReader(out))

	var curPath string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			curPath = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			if ref == "refs/heads/"+branch {
				return curPath, true, nil
			}
		case line == "":
			curPath = ""
		}
	}
	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("scan worktree list: %w", err)
	}
	return "", false, nil
}

// AddWorktree runs `git worktree add <path> <branch>` against the bare repo.
func AddWorktree(gitDir, path, branch string) error {
	_, err := RunGitDir(gitDir, "worktree", "add", path, branch)
	return err
}
