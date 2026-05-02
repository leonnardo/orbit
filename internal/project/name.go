// Package project handles project name derivation and validation.
package project

import (
	"fmt"
	"regexp"
	"strings"
)

var nameRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// Validate checks that name is a usable orbit project name.
func Validate(name string) error {
	if name == "" {
		return fmt.Errorf("project name is empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("project name cannot be %q", name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("project name cannot contain path separators: %q", name)
	}
	if !nameRE.MatchString(name) {
		return fmt.Errorf("project name has invalid characters (allowed: A-Z a-z 0-9 _ . -): %q", name)
	}
	return nil
}

// DeriveFromURL extracts the project name from a Git URL or local path.
//
// Examples:
//
//	https://github.com/x/repo-test       -> repo-test
//	https://github.com/x/repo-test.git   -> repo-test
//	git@github.com:x/repo-test.git       -> repo-test
//	ssh://git@example.com/x/repo-test    -> repo-test
//	/path/to/repo-test                   -> repo-test
//	/path/to/repo-test.git/              -> repo-test
func DeriveFromURL(url string) (string, error) {
	s := strings.TrimSpace(url)
	if s == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Strip trailing slashes.
	s = strings.TrimRight(s, "/")

	// Find the last segment, treating both '/' and ':' as separators
	// (':' covers scp-like URLs like git@host:org/repo.git).
	idx := strings.LastIndexAny(s, "/:")
	if idx >= 0 {
		s = s[idx+1:]
	}

	// Strip .git suffix.
	s = strings.TrimSuffix(s, ".git")

	if err := Validate(s); err != nil {
		return "", fmt.Errorf("cannot derive project name from %q: %w", url, err)
	}
	return s, nil
}
