package hub

import (
	"path/filepath"
	"testing"
)

func TestResolveWorktreePath(t *testing.T) {
	hub := "/Users/me/workspace/repo-test"

	tests := []struct {
		name     string
		slug     string
		userPath string
		want     string
		wantErr  bool
	}{
		{"omitted", "feature-login", "", filepath.Join(hub, "feature-login"), false},
		{"relative", "x", "worktrees/foo", filepath.Join(hub, "worktrees", "foo"), false},
		{"absolute inside", "x", filepath.Join(hub, "foo"), filepath.Join(hub, "foo"), false},
		{"absolute outside", "x", "/tmp/foo", "", true},
		{"escapes via ..", "x", "../foo", "", true},
		{"hub root itself", "x", ".", "", true},
		{"deep relative ok", "x", "a/b/c", filepath.Join(hub, "a", "b", "c"), false},
	}
	for _, tt := range tests {
		got, err := ResolveWorktreePath(hub, tt.slug, tt.userPath)
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: got %q; want error", tt.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: got %q; want %q", tt.name, got, tt.want)
		}
	}
}
