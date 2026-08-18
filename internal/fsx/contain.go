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
	// why: the file itself may not exist yet, so a failed EvalSymlinks is not
	// fatal here - the containment test below still runs on the cleaned path.
	if realPath, evalErr := filepath.EvalSymlinks(absPath); evalErr == nil {
		absPath = realPath
	}

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrOutsideRoot, path)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q", ErrOutsideRoot, path)
	}
	return absPath, nil
}
