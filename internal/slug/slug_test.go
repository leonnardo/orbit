package slug

import "testing"

func TestBranch(t *testing.T) {
	cases := map[string]string{
		"feature/login":    "feature-login",
		"fix/foo_bar":      "fix-foo_bar",
		"renovate/go-1.23": "renovate-go-1.23",
		"main":             "main",
		"a/b/c":            "a-b-c",
		"weird name!":      "weirdname",
	}
	for in, want := range cases {
		if got := Branch(in); got != want {
			t.Errorf("Branch(%q) = %q; want %q", in, got, want)
		}
	}
}
