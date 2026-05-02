package commands

import (
	"strings"
	"testing"

	"github.com/leonnardo/orbit/internal/git"
)

func TestRenderPath(t *testing.T) {
	cases := []struct {
		p, home, want string
	}{
		{"/Users/me/workspace/repo/main", "/Users/me", "~/workspace/repo/main"},
		{"/Users/me", "/Users/me", "~"},
		{"/elsewhere/x", "/Users/me", "/elsewhere/x"},
		{"/Users/meeting/wt", "/Users/me", "/Users/meeting/wt"}, // prefix-only must not match
		{"/abs/path", "", "/abs/path"},
	}
	for _, c := range cases {
		if got := renderPath(c.p, c.home); got != c.want {
			t.Errorf("renderPath(%q, %q) = %q, want %q", c.p, c.home, got, c.want)
		}
	}
}

func TestFormatList(t *testing.T) {
	hub := "/Users/me/workspace/repo-test"
	home := "/Users/me"
	entries := []git.WorktreeEntry{
		{Path: "/state/orbit/repos/repo-test", Bare: true},
		{Path: "/Users/me/workspace/repo-test/main", Branch: "refs/heads/main"},
		{Path: "/Users/me/workspace/repo-test/feature-login", Branch: "refs/heads/feature/login"},
		{Path: "/Users/me/workspace/repo-test/fix-bug", Branch: "refs/heads/fix-bug"},
		{Path: "/Users/me/elsewhere/external", Branch: "refs/heads/ext"}, // outside hub: must be skipped
	}
	got := formatList("repo-test", "https://github.com/leonnardo/repo-test", hub, entries, home, "")

	wantHeader := "repo-test  (https://github.com/leonnardo/repo-test)"
	if !strings.HasPrefix(got, wantHeader+"\n") {
		t.Errorf("missing header %q in:\n%s", wantHeader, got)
	}

	// Sorted: feature-login, fix-bug, main.
	// name column padded to len("feature-login") = 13.
	// branch column padded to len("feature/login") = 13.
	// No cwd → every row gets a leading space.
	wantLines := []string{
		"  feature-login  feature/login  ~/workspace/repo-test/feature-login",
		"  fix-bug        fix-bug        ~/workspace/repo-test/fix-bug",
		"  main           main           ~/workspace/repo-test/main",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}

	if strings.Contains(got, "external") {
		t.Errorf("external worktree should not appear:\n%s", got)
	}
	if strings.Contains(got, "/state/orbit/repos") {
		t.Errorf("bare entry should not appear:\n%s", got)
	}

	// Verify bytes-exact ordering.
	wantOrder := wantHeader + "\n" + wantLines[0] + "\n" + wantLines[1] + "\n" + wantLines[2] + "\n"
	if got != wantOrder {
		t.Errorf("output mismatch:\nwant:\n%s\ngot:\n%s", wantOrder, got)
	}
}

func TestFormatList_Detached(t *testing.T) {
	hub := "/Users/me/workspace/repo-test"
	entries := []git.WorktreeEntry{
		{Path: "/Users/me/workspace/repo-test/main", Branch: "refs/heads/main"},
		{Path: "/Users/me/workspace/repo-test/detached-wt", Detached: true},
	}
	got := formatList("repo-test", "", hub, entries, "/Users/me", "")

	// name column padded to len("detached-wt") = 11.
	// branch column padded to len("(detached)") = 10.
	wantLines := []string{
		"  detached-wt  (detached)  ~/workspace/repo-test/detached-wt",
		"  main         main        ~/workspace/repo-test/main",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}
}

func TestFormatList_CwdMarker(t *testing.T) {
	hub := "/Users/me/workspace/repo-test"
	home := "/Users/me"
	entries := []git.WorktreeEntry{
		{Path: "/Users/me/workspace/repo-test/main", Branch: "refs/heads/main"},
		{Path: "/Users/me/workspace/repo-test/feature-login", Branch: "refs/heads/feature/login"},
	}

	// cwd inside the "main" worktree → only main is marked with "*".
	got := formatList("repo-test", "", hub, entries, home, "/Users/me/workspace/repo-test/main/internal/cmd")
	wantLines := []string{
		"  feature-login  feature/login  ~/workspace/repo-test/feature-login",
		"* main           main           ~/workspace/repo-test/main",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}
	if strings.Contains(got, "* feature-login") {
		t.Errorf("feature-login should not be marked with *:\n%s", got)
	}

	// cwd outside any worktree (e.g. hub root) → no row gets "*".
	got2 := formatList("repo-test", "", hub, entries, home, hub)
	if strings.Contains(got2, "*") {
		t.Errorf("no row should be marked when cwd is outside every worktree:\n%s", got2)
	}

	// Empty cwd → no row gets "*".
	got3 := formatList("repo-test", "", hub, entries, home, "")
	if strings.Contains(got3, "*") {
		t.Errorf("no row should be marked when cwd is empty:\n%s", got3)
	}
}

func TestFormatList_Empty(t *testing.T) {
	hub := "/h"
	got := formatList("proj", "https://example.com/proj", hub, []git.WorktreeEntry{
		{Path: "/state/bare", Bare: true},
	}, "", "")

	wantHeader := "proj  (https://example.com/proj)\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("missing header in:\n%s", got)
	}
	if !strings.Contains(got, "no worktrees yet") {
		t.Errorf("missing empty hint in:\n%s", got)
	}
}

func TestFormatList_NoRemote(t *testing.T) {
	got := formatList("proj", "", "/h", nil, "", "")
	if !strings.HasPrefix(got, "proj\n") {
		t.Errorf("expected bare project header, got:\n%s", got)
	}
}
