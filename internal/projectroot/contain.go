package projectroot

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrOutsideRoot = errors.New("path resolves outside the project root")

func ResolveInside(root, path string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", path, err)
	}
	if realRoot, evalErr := filepath.EvalSymlinks(absRoot); evalErr == nil {
		absRoot = realRoot
	}
	// Both sides must resolve: a link is how a contained-looking path escapes.
	absPath = resolveExisting(absPath)

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrOutsideRoot, path)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q", ErrOutsideRoot, path)
	}
	return absPath, nil
}

func resolveExisting(path string) string {
	remainder := ""
	// EvalSymlinks fails on a not-yet-created leaf, so resolve the deepest ancestor
	// that exists and append the rest; darwin maps /var to /private/var.
	for current := path; ; {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			return filepath.Join(resolved, remainder)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		remainder = filepath.Join(filepath.Base(current), remainder)
		current = parent
	}
}
