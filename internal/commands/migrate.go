package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leonnardo/orbit/internal/git"
	"github.com/leonnardo/orbit/internal/hub"
	"github.com/leonnardo/orbit/internal/project"
	"github.com/leonnardo/orbit/internal/slug"
	"github.com/leonnardo/orbit/internal/state"
)

const migrateUsage = `usage: orbit migrate [--name <project>]`

// migrateBranch is a local head with its current SHA, captured during preflight.
type migrateBranch struct {
	name string
	sha  string
}

// Migrate adopts an existing standalone Git clone in the current directory as
// an orbit hub.
//
// It runs a long list of preflight checks (clean tree, no stash, all branches
// pushed, no path collisions, etc.), and only then renames the current
// directory aside as a backup, creates a fresh bare in the orbit state dir,
// re-fetches from origin, writes the hub config, and recreates each local
// branch as a worktree.
//
// The original .git directory (hooks, info/exclude, reflog, stash, config
// extras) is NOT carried into the new bare — it lives in the backup directory
// for the user to inspect before deleting.
func Migrate(args []string) error {
	name, err := parseMigrateArgs(args)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}
	origDir, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}

	// Preflight 1: cwd has a .git directory (not a gitlink file).
	gitInfo, err := os.Stat(filepath.Join(origDir, ".git"))
	if err != nil {
		return fmt.Errorf("not a git repository: no .git found at %s\n  run `orbit migrate` from inside an existing standalone git clone", origDir)
	}
	if !gitInfo.IsDir() {
		return fmt.Errorf(".git at %s is a file, not a directory\n  this looks like a worktree (gitlink), not a standalone clone — orbit migrate adopts standalone clones only", origDir)
	}

	// Preflight 10: parent must exist; cwd must not be the filesystem root.
	hubRoot := filepath.Dir(origDir)
	if hubRoot == origDir {
		return fmt.Errorf("cannot migrate %s: it has no parent directory\n  orbit places worktrees as siblings of the original clone, so the parent must exist", origDir)
	}
	if info, err := os.Stat(hubRoot); err != nil || !info.IsDir() {
		return fmt.Errorf("parent directory does not exist or is not a directory: %s", hubRoot)
	}

	// Preflight 2: working tree is clean.
	statusOut, err := git.RunInDir(origDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("could not check working tree status: %w", err)
	}
	if strings.TrimSpace(statusOut) != "" {
		return fmt.Errorf("working tree is not clean; orbit migrate refuses to touch a dirty clone\n  commit, stash, or discard your changes first\n  output of `git status --porcelain`:\n%s", indentBlock(statusOut))
	}

	// Preflight 3: no stash entries.
	stashOut, err := git.RunInDir(origDir, "stash", "list")
	if err != nil {
		return fmt.Errorf("could not list stash entries: %w", err)
	}
	if strings.TrimSpace(stashOut) != "" {
		return fmt.Errorf("repository has stash entries; orbit migrate cannot preserve them\n  pop or drop them first (`git stash pop` / `git stash drop`)\n  output of `git stash list`:\n%s", indentBlock(stashOut))
	}

	// Preflight 4: origin remote with non-empty URL.
	remoteURL, err := git.RunInDir(origDir, "remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("no remote named `origin` (or it has no URL)\n  add one first: `git remote add origin <url>`")
	}
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return fmt.Errorf("remote `origin` has an empty URL\n  set one with `git remote set-url origin <url>`")
	}

	// Preflight 5: derive or validate the project name.
	if name == "" {
		name, err = project.DeriveFromURL(remoteURL)
		if err != nil {
			return fmt.Errorf("%w\n  pass an explicit name with `orbit migrate --name <project>`", err)
		}
	} else {
		if err := project.Validate(name); err != nil {
			return err
		}
	}

	// Preflight 6: bare path must not already exist.
	barePath, err := state.BarePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(barePath); err == nil {
		return fmt.Errorf("project %q already exists in orbit state: %s\n  rename or delete that bare repo, or pass `--name <other>` to migrate under a different project name", name, barePath)
	}

	// Preflight 7: parent must not already be a hub.
	parentMarker := filepath.Join(hubRoot, hub.ConfigFilename)
	if _, err := os.Stat(parentMarker); err == nil {
		return fmt.Errorf("parent directory %s is already an orbit hub (has %s)\n  this clone cannot be adopted into an existing hub", hubRoot, hub.ConfigFilename)
	}

	// Collect local branches with SHAs (used by preflights 8 and 9, and by
	// execution).
	branches, err := listLocalBranches(origDir)
	if err != nil {
		return err
	}

	// Preflight 9: every local branch must be pushed (matching SHA on origin).
	if msg := checkAllPushed(origDir, branches); msg != "" {
		return errors.New(msg)
	}

	// Determine current branch (empty if HEAD is detached).
	currentBranch := ""
	if out, err := git.RunInDir(origDir, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		currentBranch = strings.TrimSpace(out)
	}

	// Preflight 8: each candidate worktree path (parent/<slug>) must not exist,
	// except for origDir itself (which we'll move aside before creating it).
	if msg := checkSlugCollisions(hubRoot, origDir, branches); msg != "" {
		return errors.New(msg)
	}

	// === All preflights passed. Mutate. ===

	timestamp := time.Now().Unix()
	backupDir := fmt.Sprintf("%s.orbit-backup-%d", origDir, timestamp)
	fmt.Fprintf(os.Stderr, "orbit: moving original clone aside\n  from: %s\n  to:   %s\n", origDir, backupDir)
	if err := os.Rename(origDir, backupDir); err != nil {
		return fmt.Errorf("rename original clone: %w\n  no changes were made", err)
	}

	// From here on, partial failures are NOT auto-rolled back. The backup is
	// the user's escape hatch; we report it in every error message.
	failPartial := func(format string, args ...any) error {
		msg := fmt.Sprintf(format, args...)
		return fmt.Errorf("%s\n  partial state — your original clone is still safe at: %s\n  inspect it and recover manually (you may need to delete the bare at %s and try again)", msg, backupDir, barePath)
	}

	if _, err := state.EnsureReposDir(); err != nil {
		return failPartial("create repos dir: %v", err)
	}

	fmt.Fprintf(os.Stderr, "orbit: creating bare repo at %s\n", barePath)
	if _, gerr := git.Run("init", "--bare", barePath); gerr != nil {
		return failPartial("git init --bare failed: %v", gerr)
	}

	if _, gerr := git.RunGitDir(barePath, "remote", "add", "origin", remoteURL); gerr != nil {
		return failPartial("configure origin remote: %v", gerr)
	}

	fmt.Fprintf(os.Stderr, "orbit: fetching from %s\n", remoteURL)
	if ferr := git.StreamGitDir(barePath, "fetch", "origin",
		"+refs/heads/*:refs/remotes/origin/*",
		"+refs/tags/*:refs/tags/*",
		"--prune"); ferr != nil {
		return failPartial("fetch from origin failed: %v", ferr)
	}

	if _, gerr := git.RunGitDir(barePath, "remote", "set-head", "origin", "-a"); gerr != nil {
		fmt.Fprintf(os.Stderr, "orbit: warning: could not set origin/HEAD: %v\n", gerr)
	}

	cfg := &hub.Config{
		Version:   1,
		Project:   name,
		Remote:    remoteURL,
		CreatedAt: time.Now().UTC(),
	}
	cfgPath := filepath.Join(hubRoot, hub.ConfigFilename)
	if werr := hub.Write(cfgPath, cfg); werr != nil {
		return failPartial("write hub config: %v", werr)
	}

	// Switch into the hub root so commands.New can detect the hub via cwd.
	if err := os.Chdir(hubRoot); err != nil {
		return failPartial("chdir to hub root %s: %v", hubRoot, err)
	}

	ordered := orderBranchesForRecreate(branches, currentBranch)

	var recreated []string
	var failures []string
	for _, b := range ordered {
		fmt.Fprintf(os.Stderr, "orbit: recreating worktree for branch %s\n", b.name)
		if err := New([]string{b.name}); err != nil {
			failures = append(failures, fmt.Sprintf("  - %s: %v", b.name, err))
			continue
		}
		recreated = append(recreated, b.name)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "orbit: migration complete")
	fmt.Fprintf(os.Stderr, "  hub:  %s\n", hubRoot)
	fmt.Fprintf(os.Stderr, "  bare: %s\n", barePath)
	if len(recreated) > 0 {
		fmt.Fprintln(os.Stderr, "  recreated worktrees:")
		for _, bn := range recreated {
			fmt.Fprintf(os.Stderr, "    %-30s -> %s\n", bn, filepath.Join(hubRoot, slug.Branch(bn)))
		}
	} else {
		fmt.Fprintln(os.Stderr, "  no worktrees recreated (no local branches to migrate)")
	}
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "  failed to recreate (best-effort; retry manually with `orbit new <branch>`):")
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, f)
		}
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  backup: %s\n", backupDir)
	fmt.Fprintln(os.Stderr, "    contains the original .git (hooks, info/exclude, reflog, etc.)")
	fmt.Fprintf(os.Stderr, "    verify the new hub works, then `rm -rf %s`\n", backupDir)
	if currentBranch != "" {
		wtPath := filepath.Join(hubRoot, slug.Branch(currentBranch))
		fmt.Fprintf(os.Stderr, "\n  your shell's cwd is now stale; cd into the new worktree:\n    cd %s\n", wtPath)
	} else {
		fmt.Fprintf(os.Stderr, "\n  your shell's cwd is now stale; cd into the hub:\n    cd %s\n", hubRoot)
	}

	return nil
}

func parseMigrateArgs(args []string) (string, error) {
	var name string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return "", errors.New(migrateUsage)
		case a == "--name":
			if i+1 >= len(args) {
				return "", fmt.Errorf("--name requires a value\n%s", migrateUsage)
			}
			name = args[i+1]
			i++
		case strings.HasPrefix(a, "--name="):
			name = strings.TrimPrefix(a, "--name=")
			if name == "" {
				return "", fmt.Errorf("--name requires a value\n%s", migrateUsage)
			}
		case strings.HasPrefix(a, "-"):
			return "", fmt.Errorf("unknown flag %q\n%s", a, migrateUsage)
		default:
			return "", fmt.Errorf("unexpected argument %q\n%s", a, migrateUsage)
		}
	}
	return name, nil
}

// listLocalBranches returns each refs/heads/X with its SHA, sorted by name.
func listLocalBranches(workdir string) ([]migrateBranch, error) {
	out, err := git.RunInDir(workdir, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads/")
	if err != nil {
		return nil, fmt.Errorf("list local branches: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	var branches []migrateBranch
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		branches = append(branches, migrateBranch{name: parts[0], sha: parts[1]})
	}
	sort.Slice(branches, func(i, j int) bool { return branches[i].name < branches[j].name })
	return branches, nil
}

// checkAllPushed verifies every local branch has a matching origin counterpart.
// Returns a multi-line error message (without a trailing newline) on failure,
// or "" on success.
func checkAllPushed(workdir string, branches []migrateBranch) string {
	var problems []string
	for _, b := range branches {
		out, err := git.RunInDir(workdir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+b.name)
		if err != nil {
			problems = append(problems, fmt.Sprintf("  - %s: no origin/%s (push it first: `git push -u origin %s`)", b.name, b.name, b.name))
			continue
		}
		originSha := strings.TrimSpace(out)
		if originSha != b.sha {
			problems = append(problems, fmt.Sprintf("  - %s: local %s differs from origin/%s %s (push or reset before migrating)", b.name, shortSHA(b.sha), b.name, shortSHA(originSha)))
		}
	}
	if len(problems) == 0 {
		return ""
	}
	return "local branches not pushed to origin (orbit migrate refetches from origin, so unpushed work would be lost):\n" +
		strings.Join(problems, "\n") +
		"\n  push the listed branches and re-run, or delete them locally if they are obsolete"
}

// checkSlugCollisions ensures parent/<slug(branch)> does not already exist for
// any local branch — except for origDir itself, which we will rename aside
// before creating any worktree.
func checkSlugCollisions(hubRoot, origDir string, branches []migrateBranch) string {
	var collisions []string
	for _, b := range branches {
		s := slug.Branch(b.name)
		if s == "" {
			continue
		}
		candidate := filepath.Join(hubRoot, s)
		candAbs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if candAbs == origDir {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			collisions = append(collisions, fmt.Sprintf("  - %s (would be the worktree path for branch %q)", candidate, b.name))
		}
	}
	if len(collisions) == 0 {
		return ""
	}
	return "worktree paths already exist in the parent directory:\n" +
		strings.Join(collisions, "\n") +
		"\n  remove or rename them, then re-run"
}

// orderBranchesForRecreate puts current first (so it gets first dibs on its
// slug-derived path when two branches collide), then the rest in the input
// order (which is already alphabetical from listLocalBranches).
func orderBranchesForRecreate(branches []migrateBranch, current string) []migrateBranch {
	if current == "" || len(branches) == 0 {
		return branches
	}
	out := make([]migrateBranch, 0, len(branches))
	var rest []migrateBranch
	for _, b := range branches {
		if b.name == current {
			out = append(out, b)
		} else {
			rest = append(rest, b)
		}
	}
	return append(out, rest...)
}

// shortSHA returns the first 7 chars of a SHA, or the SHA itself if shorter.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// indentBlock indents every line of s with two spaces (for embedding command
// output inside a multi-line error message).
func indentBlock(s string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = "    " + ln
	}
	return strings.Join(lines, "\n")
}
