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
	got := formatList("repo-test", "https://github.com/leonnardo/repo-test", hub, entries, home)

	wantHeader := "repo-test  (https://github.com/leonnardo/repo-test)"
	if !strings.HasPrefix(got, wantHeader+"\n") {
		t.Errorf("missing header %q in:\n%s", wantHeader, got)
	}

	// Sorted: feature-login, fix-bug, main; padded to len("feature-login") = 13.
	wantLines := []string{
		"  feature-login  ~/workspace/repo-test/feature-login",
		"  fix-bug        ~/workspace/repo-test/fix-bug",
		"  main           ~/workspace/repo-test/main",
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

func TestFormatList_Empty(t *testing.T) {
	hub := "/h"
	got := formatList("proj", "https://example.com/proj", hub, []git.WorktreeEntry{
		{Path: "/state/bare", Bare: true},
	}, "")

	wantHeader := "proj  (https://example.com/proj)\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("missing header in:\n%s", got)
	}
	if !strings.Contains(got, "no worktrees yet") {
		t.Errorf("missing empty hint in:\n%s", got)
	}
}

func TestFormatList_NoRemote(t *testing.T) {
	got := formatList("proj", "", "/h", nil, "")
	if !strings.HasPrefix(got, "proj\n") {
		t.Errorf("expected bare project header, got:\n%s", got)
	}
}
