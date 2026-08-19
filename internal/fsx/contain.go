package fsx

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrOutsideRoot is returned when a path resolves outside the root it must stay under.
var ErrOutsideRoot = errors.New("path resolves outside the project root")

// ResolveInside returns path's absolute, symlink-free form, failing when it is
// not under root.
// why: both sides resolve - a link is how a contained-looking path stops being
// one, and macOS maps /tmp to /private/tmp, so resolving one side only gets
// both answers wrong.
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

// resolveExisting returns path with its deepest EXISTING ancestor resolved
// through symlinks and the not-yet-created remainder appended back.
// why: EvalSymlinks fails when the leaf does not exist, the normal case for a
// file about to be written. Resolving only the root then compares two spellings
// of one tree - on darwin (/var -> /private/var) every new file reads as escape.
func resolveExisting(path string) string {
	remainder := ""
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
