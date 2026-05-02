// Package slug derives filesystem-friendly names from branch names.
package slug

import "strings"

// Branch converts a branch name into a worktree directory slug.
//
// Rules: '/' becomes '-', other allowed chars (letters, digits, '_', '.', '-')
// are preserved. Disallowed chars are dropped. The MVP keeps this conservative
// because git already restricts branch names heavily.
func Branch(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r == '/':
			b.WriteByte('-')
		case r == '_' || r == '.' || r == '-':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}
