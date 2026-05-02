package hub

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveWorktreePath resolves the final worktree path inside the hub.
//
// If userPath is empty, returns hubRoot/slug.
// If userPath is relative, joins with hubRoot.
// If userPath is absolute, uses as-is.
//
// In all cases, the resolved path must be strictly inside hubRoot.
func ResolveWorktreePath(hubRoot, slug, userPath string) (string, error) {
	hubAbs, err := filepath.Abs(hubRoot)
	if err != nil {
		return "", fmt.Errorf("resolve hub root: %w", err)
	}

	var raw string
	switch {
	case userPath == "":
		raw = filepath.Join(hubAbs, slug)
	case filepath.IsAbs(userPath):
		raw = userPath
	default:
		raw = filepath.Join(hubAbs, userPath)
	}

	clean := filepath.Clean(raw)

	// Must be strictly inside the hub: prefix match plus separator.
	if clean == hubAbs {
		return "", fmt.Errorf("worktree path cannot be the hub root itself: %s", clean)
	}
	if !strings.HasPrefix(clean, hubAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("worktree path must be inside the project hub\n  hub:  %s\n  path: %s", hubAbs, clean)
	}
	return clean, nil
}
