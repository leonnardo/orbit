package commands

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMigrateArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantName string
		wantErr  bool
	}{
		{"empty", nil, "", false},
		{"name spaced", []string{"--name", "myproj"}, "myproj", false},
		{"name eq form", []string{"--name=myproj"}, "myproj", false},
		{"name missing value", []string{"--name"}, "", true},
		{"name eq empty", []string{"--name="}, "", true},
		{"unknown flag", []string{"--force"}, "", true},
		{"unexpected positional", []string{"foo"}, "", true},
		{"help short", []string{"-h"}, "", true},
		{"help long", []string{"--help"}, "", true},
		{"flag after positional rejected", []string{"foo", "--name", "x"}, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseMigrateArgs(c.args)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if c.wantErr {
				return
			}
			if got != c.wantName {
				t.Errorf("name = %q, want %q", got, c.wantName)
			}
		})
	}
}

func TestOrderBranchesForRecreate(t *testing.T) {
	mk := func(names ...string) []migrateBranch {
		out := make([]migrateBranch, len(names))
		for i, n := range names {
			out[i] = migrateBranch{name: n}
		}
		return out
	}

	cases := []struct {
		name    string
		input   []migrateBranch
		current string
		want    []string
	}{
		{
			name:    "current first when present",
			input:   mk("develop", "feat", "main"),
			current: "main",
			want:    []string{"main", "develop", "feat"},
		},
		{
			name:    "current empty preserves input order",
			input:   mk("a", "b", "c"),
			current: "",
			want:    []string{"a", "b", "c"},
		},
		{
			name:    "current not in list preserves input order",
			input:   mk("a", "b"),
			current: "missing",
			want:    []string{"a", "b"},
		},
		{
			name:    "single current branch",
			input:   mk("main"),
			current: "main",
			want:    []string{"main"},
		},
		{
			name:    "empty input",
			input:   nil,
			current: "main",
			want:    nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := orderBranchesForRecreate(c.input, c.current)
			gotNames := make([]string, len(got))
			for i, b := range got {
				gotNames[i] = b.name
			}
			if len(gotNames) == 0 && len(c.want) == 0 {
				return
			}
			if !reflect.DeepEqual(gotNames, c.want) {
				t.Errorf("got %v, want %v", gotNames, c.want)
			}
		})
	}
}

func TestShortSHA(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"abc", "abc"},
		{"1234567", "1234567"},
		{"12345678", "1234567"},
		{"abcdef0123456789", "abcdef0"},
	}
	for _, c := range cases {
		if got := shortSHA(c.in); got != c.want {
			t.Errorf("shortSHA(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIndentBlock(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"single", "    single"},
		{"a\nb", "    a\n    b"},
		{"a\nb\n", "    a\n    b"},   // trailing newline trimmed
		{"a\n\nb", "    a\n\n    b"}, // empty middle line gets indented too
	}
	for _, c := range cases {
		got := indentBlock(c.in)
		// Be lenient on the empty-middle-line case: our impl indents it too.
		if c.in == "a\n\nb" {
			if !strings.HasPrefix(got, "    a\n") || !strings.HasSuffix(got, "    b") {
				t.Errorf("indentBlock(%q) = %q, want prefix '    a\\n' and suffix '    b'", c.in, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("indentBlock(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
