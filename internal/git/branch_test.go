package git

import "testing"

func TestParseWorktreeList(t *testing.T) {
	const sample = `worktree /repos/orbit
HEAD 0000000000000000000000000000000000000000
bare

worktree /work/orbit/main
HEAD 1111111111111111111111111111111111111111
branch refs/heads/main

worktree /work/orbit/feat
HEAD 2222222222222222222222222222222222222222
detached

worktree /work/orbit/locked-one
HEAD 3333333333333333333333333333333333333333
branch refs/heads/feat/locked
locked some reason

`

	got, err := parseWorktreeList(sample)
	if err != nil {
		t.Fatalf("parseWorktreeList: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 entries, got %d: %+v", len(got), got)
	}

	cases := []struct {
		idx      int
		path     string
		branch   string
		bare     bool
		detached bool
		locked   bool
	}{
		{0, "/repos/orbit", "", true, false, false},
		{1, "/work/orbit/main", "refs/heads/main", false, false, false},
		{2, "/work/orbit/feat", "", false, true, false},
		{3, "/work/orbit/locked-one", "refs/heads/feat/locked", false, false, true},
	}
	for _, c := range cases {
		e := got[c.idx]
		if e.Path != c.path || e.Branch != c.branch || e.Bare != c.bare || e.Detached != c.detached || e.Locked != c.locked {
			t.Errorf("entry %d mismatch:\n  got:  %+v\n  want: path=%q branch=%q bare=%v detached=%v locked=%v",
				c.idx, e, c.path, c.branch, c.bare, c.detached, c.locked)
		}
	}
}

func TestWorktreeEntry_BranchName(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feat/login", "feat/login"},
		{"", ""},
		{"refs/tags/v1", ""}, // not a branch ref
	}
	for _, c := range cases {
		got := WorktreeEntry{Branch: c.ref}.BranchName()
		if got != c.want {
			t.Errorf("BranchName(%q) = %q, want %q", c.ref, got, c.want)
		}
	}
}
