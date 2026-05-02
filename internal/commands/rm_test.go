package commands

import (
	"path/filepath"
	"testing"

	"github.com/leonnardo/orbit/internal/git"
)

func TestParseRmArgs(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantTarget  string
		wantDelete  bool
		wantErr     bool
	}{
		{"no args", []string{}, "", false, true},
		{"only flag", []string{"--delete-branch"}, "", false, true},
		{"target only", []string{"feat"}, "feat", false, false},
		{"target + flag", []string{"feat", "--delete-branch"}, "feat", true, false},
		{"flag + target", []string{"--delete-branch", "feat"}, "feat", true, false},
		{"two targets", []string{"a", "b"}, "", false, true},
		{"unknown flag", []string{"feat", "--force"}, "", false, true},
		{"help short", []string{"-h"}, "", false, true},
		{"abs path", []string{"/abs/path/wt"}, "/abs/path/wt", false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tgt, del, err := parseRmArgs(c.args)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if c.wantErr {
				return
			}
			if tgt != c.wantTarget {
				t.Errorf("target = %q, want %q", tgt, c.wantTarget)
			}
			if del != c.wantDelete {
				t.Errorf("delete = %v, want %v", del, c.wantDelete)
			}
		})
	}
}

func TestIsExplicitPath(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"feat", false},
		{"feat-login", false},
		{"feat/login", true},
		{"./feat", true},
		{"../feat", true},
		{".", true},
		{"..", true},
		{"/abs/path", true},
		{`win\path`, true},
	}
	for _, c := range cases {
		if got := isExplicitPath(c.in); got != c.want {
			t.Errorf("isExplicitPath(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestResolveRmTarget(t *testing.T) {
	// Use synthetic absolute paths under a fake hub root. EvalSymlinks falls
	// back to Clean for non-existent paths, so all comparisons are lexical.
	hub := "/h"
	entries := []git.WorktreeEntry{
		{Path: "/bare", Bare: true},
		{Path: "/h/main", Branch: "refs/heads/main"},
		{Path: "/h/feat-login", Branch: "refs/heads/feat/login"},
		{Path: "/h/sub/dir", Branch: "refs/heads/sub"},
		{Path: "/elsewhere/wt", Branch: "refs/heads/external"}, // outside hub — must be ignored
	}

	cases := []struct {
		name      string
		target    string
		cwd       string
		wantPath  string // empty = expect error
	}{
		{"basename matches hub-local", "main", "/anywhere", "/h/main"},
		{"basename with slug", "feat-login", "/anywhere", "/h/feat-login"},
		{"basename no match", "nope", "/anywhere", ""},
		{"basename of external worktree is ignored", "wt", "/anywhere", ""},
		{"abs path inside hub", "/h/main", "/anywhere", "/h/main"},
		{"abs path outside hub is rejected", "/elsewhere/wt", "/anywhere", ""},
		{"relative dot path from cwd", "./feat-login", "/h", "/h/feat-login"},
		{"relative parent path from sub", "../main", "/h/sub", "/h/main"},
		{"relative path with slash interpreted as path not basename", "sub/dir", "/h", "/h/sub/dir"},
		{"basename matches cwd-named dir but resolves to hub", "main", "/h/main", "/h/main"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveRmTarget(c.target, c.cwd, hub, entries)
			if c.wantPath == "" {
				if err == nil {
					t.Fatalf("want error, got entry %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filepath.Clean(got.Path) != filepath.Clean(c.wantPath) {
				t.Errorf("path = %q, want %q", got.Path, c.wantPath)
			}
		})
	}
}

func TestIsInside(t *testing.T) {
	cases := []struct {
		child, parent string
		want          bool
	}{
		{"/h/wt", "/h/wt", true},
		{"/h/wt/sub", "/h/wt", true},
		{"/h/wt-other", "/h/wt", false}, // prefix-only must not match
		{"/h", "/h/wt", false},
		{"/elsewhere", "/h/wt", false},
	}
	for _, c := range cases {
		if got := isInside(c.child, c.parent); got != c.want {
			t.Errorf("isInside(%q, %q) = %v, want %v", c.child, c.parent, got, c.want)
		}
	}
}
