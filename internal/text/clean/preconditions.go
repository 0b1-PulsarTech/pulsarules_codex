package clean

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

// read applies every precondition and returns the file's text, or a Report whose
// SkipReason names the gate that refused it.
//
// why: a refusal is data, not an error - the hook must stay silent and never
// block a turn, and the command wants to say what it skipped.
func (c *Cleaner) read(path string) (Report, string, error) {
	report := Report{Path: path}
	if c.root == "" {
		report.SkipReason = "no project root"
		return report, "", nil
	}
	if !eligible[strings.ToLower(filepath.Ext(path))] {
		report.SkipReason = "extension is not .go or .md"
		return report, "", nil
	}

	resolved, err := fsx.ResolveInside(c.root, path)
	if err != nil {
		if errors.Is(err, fsx.ErrOutsideRoot) {
			report.SkipReason = "outside the project root"
			return report, "", nil
		}
		return report, "", err
	}
	report.Path = resolved

	if isFixture(c.root, resolved) {
		report.SkipReason = "under testdata"
		return report, "", nil
	}
	if reason := statGate(resolved); reason != "" {
		report.SkipReason = reason
		return report, "", nil
	}

	src, err := os.ReadFile(resolved) //nolint:gosec // resolved by ResolveInside.
	if err != nil {
		return report, "", fmt.Errorf("read %q: %w", path, err)
	}
	if !utf8.Valid(src) {
		// why: Scan ranges over runes, so invalid bytes decode to U+FFFD and a
		// splice would land at the wrong offset - refuse rather than corrupt.
		report.SkipReason = "not valid UTF-8"
		return report, "", nil
	}
	return report, string(src), nil
}

func statGate(path string) string {
	stat, err := os.Lstat(path)
	switch {
	case err != nil:
		return "cannot stat"
	case stat.Mode()&fs.ModeSymlink != 0:
		return "is a symlink"
	case !stat.Mode().IsRegular():
		return "not a regular file"
	case stat.Size() > maxBytes:
		return "larger than the size cap"
	}
	return ""
}

// why: testdata holds deliberately non-conforming fixtures - the same carve-out
// core.walkSkipDirs makes for the analyzer walk, repeated here because the
// changed-file path has no such filter.
func isFixture(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return slices.Contains(strings.Split(filepath.ToSlash(rel), "/"), "testdata")
}
