package commands

import (
	"strings"
	"testing"

	"github.com/leonnardo/orbit/internal/git"
)

func TestFormatSignals(t *testing.T) {
	cases := []struct {
		name string
		e    git.WorktreeEntry
		st   git.WorktreeStatus
		want string
	}{
		{name: "clean", want: "✓"},
		{name: "changed", st: git.WorktreeStatus{Changed: true}, want: "●"},
		{name: "untracked", st: git.WorktreeStatus{Untracked: true}, want: "?"},
		{name: "conflict", st: git.WorktreeStatus{Conflict: true}, want: "!"},
		{name: "ahead", st: git.WorktreeStatus{Ahead: true}, want: "↑"},
		{name: "behind", st: git.WorktreeStatus{Behind: true}, want: "↓"},
		{name: "combined", e: git.WorktreeEntry{Locked: true, Detached: true}, st: git.WorktreeStatus{Changed: true, Untracked: true, Ahead: true, Behind: true}, want: "🔒◆●?↑↓"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatSignals(c.e, c.st); got != c.want {
				t.Fatalf("formatSignals() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFormatList(t *testing.T) {
	hub := "/Users/me/workspace/repo-test"
	rows := []listRow{
		{branch: "feature/login", status: "●?", path: "feature-login", commit: "2222222", message: "Polish list output", absPath: hub + "/feature-login"},
		{branch: "fix-bug", status: "✓", path: "fix-bug", commit: "3333333", message: "Fix bug", absPath: hub + "/fix-bug"},
		{branch: "main", status: "✓", path: "main", commit: "1111111", message: "Initial commit", absPath: hub + "/main"},
	}
	got := formatList(hub, rows, "")

	want := strings.Join([]string{
		"BRANCH         STATUS  PATH           COMMIT   MESSAGE",
		"feature/login  ●?      feature-login  2222222  Polish list output",
		"fix-bug        ✓       fix-bug        3333333  Fix bug",
		"main           ✓       main           1111111  Initial commit",
		"",
	}, "\n")
	if got != want {
		t.Errorf("output mismatch:\nwant:\n%s\ngot:\n%s", want, got)
	}
	if strings.Contains(got, "* main") {
		t.Errorf("hub/outside output must not mark current rows with '*':\n%s", got)
	}
}

func TestFormatList_CurrentInsideWorktree(t *testing.T) {
	hub := "/Users/me/workspace/repo-test"
	rows := []listRow{
		{branch: "feature/login", status: "✓", path: "feature-login", commit: "2222222", message: "Polish list output", absPath: hub + "/feature-login"},
		{branch: "main", status: "✓", path: "main", commit: "1111111", message: "Initial commit", absPath: hub + "/main"},
	}

	got := formatList(hub, rows, hub+"/main/internal/cmd")
	if strings.Contains(got, "current:") {
		t.Errorf("list output should not print current header:\n%s", got)
	}
	if !strings.Contains(got, "* main           ✓       main           1111111  Initial commit\n") {
		t.Errorf("missing current worktree marker:\n%s", got)
	}
	if !strings.Contains(got, "  feature/login  ✓       feature-login  2222222  Polish list output\n") {
		t.Errorf("non-current rows should reserve marker column:\n%s", got)
	}

	got = formatList(hub, rows, hub)
	if strings.Contains(got, "current:") {
		t.Errorf("hub root should not print current worktree:\n%s", got)
	}
	if strings.Contains(got, "*") {
		t.Errorf("hub root should not mark any row:\n%s", got)
	}

	got = formatList(hub, rows, "")
	if strings.Contains(got, "current:") {
		t.Errorf("empty cwd should not print current worktree:\n%s", got)
	}
	if strings.Contains(got, "*") {
		t.Errorf("empty cwd should not mark any row:\n%s", got)
	}
}

func TestFormatList_EmptyHasNoHeader(t *testing.T) {
	got := formatList("/h", nil, "")
	if strings.Contains(got, "BRANCH") || strings.Contains(got, "STATUS") || strings.Contains(got, "MESSAGE") {
		t.Errorf("empty hub must not print column headers, got:\n%s", got)
	}
	if !strings.Contains(got, "no worktrees yet") {
		t.Errorf("missing empty hint in:\n%s", got)
	}
}

func TestFormatList_NoProjectHeader(t *testing.T) {
	got := formatList("/h", []listRow{
		{branch: "main", status: "✓", path: "main", commit: "1111111", message: "Initial commit", absPath: "/h/main"},
	}, "")
	if strings.Contains(got, "proj") || strings.Contains(got, "https://") {
		t.Errorf("list output must not print project or remote header, got:\n%s", got)
	}
	if !strings.HasPrefix(got, "BRANCH") {
		t.Errorf("expected output to start with table header, got:\n%s", got)
	}
}
