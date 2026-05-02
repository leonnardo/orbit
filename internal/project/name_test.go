package project

import "testing"

func TestDeriveFromURL(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"https://github.com/x/repo-test", "repo-test", false},
		{"https://github.com/x/repo-test.git", "repo-test", false},
		{"git@github.com:x/repo-test.git", "repo-test", false},
		{"ssh://git@example.com/x/repo-test", "repo-test", false},
		{"ssh://git@example.com/team/repo-test.git", "repo-test", false},
		{"/path/to/repo-test", "repo-test", false},
		{"/path/to/repo-test.git/", "repo-test", false},
		{"repo-test", "repo-test", false},
		{"", "", true},
		{"   ", "", true},
		{"foo bar", "", true},
	}
	for _, tt := range tests {
		got, err := DeriveFromURL(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("DeriveFromURL(%q) = %q; want error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("DeriveFromURL(%q) error = %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("DeriveFromURL(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidate(t *testing.T) {
	ok := []string{"repo-test", "api", "api.v2", "my_repo", "client-foo", "X"}
	bad := []string{"", ".", "..", "../repo", "foo/bar", "repo test", "repo:foo", "repo$x"}

	for _, n := range ok {
		if err := Validate(n); err != nil {
			t.Errorf("Validate(%q) unexpected error: %v", n, err)
		}
	}
	for _, n := range bad {
		if err := Validate(n); err == nil {
			t.Errorf("Validate(%q) = nil; want error", n)
		}
	}
}
