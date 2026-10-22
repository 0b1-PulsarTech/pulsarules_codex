package cleaner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/projectroot"
)

func (scrub *Cleaner) read(path string) (Report, string, error) {
	report := Report{Path: path}
	if scrub.root == "" {
		// A refusal is data, not an error: the hook must stay silent and never block a turn.
		report.SkipReason = "no project root"
		return report, "", nil
	}
	if !eligibleExt(strings.ToLower(filepath.Ext(path))) {
		report.SkipReason = "extension is not .go or .md"
		return report, "", nil
	}

	resolved, err := projectroot.ResolveInside(scrub.root, path)
	if err != nil {
		if errors.Is(err, projectroot.ErrOutsideRoot) {
			report.SkipReason = "outside the project root"
			return report, "", nil
		}
		return report, "", err
	}
	report.Path = resolved

	if isFixture(scrub.root, resolved) {
		report.SkipReason = "under testdata"
		return report, "", nil
	}
	if reason := projectroot.StatGate(resolved); reason != "" {
		report.SkipReason = reason
		return report, "", nil
	}

	src, err := os.ReadFile(resolved) //nolint:gosec // resolved by ResolveInside.
	if err != nil {
		return report, "", fmt.Errorf("read %q: %w", path, err)
	}
	if !utf8.Valid(src) {
		// Scan ranges over runes, so invalid bytes decode to U+FFFD and a
		// splice would land at the wrong offset, so refuse rather than corrupt.
		report.SkipReason = "not valid UTF-8"
		return report, "", nil
	}
	return report, string(src), nil
}

func isFixture(root, path string) bool {
	// Testdata fixtures are non-conforming by design; the changed-file path has no walk filter.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	// The path arrives absolute and symlink-resolved, so the root must be too.
	if realRoot, evalErr := filepath.EvalSymlinks(absRoot); evalErr == nil {
		absRoot = realRoot
	}
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return false
	}
	return slices.Contains(strings.Split(filepath.ToSlash(rel), "/"), "testdata")
}

func Handles(ext string) bool { return eligibleExt(strings.ToLower(ext)) }

func eligibleExt(ext string) bool {
	// The mutation blast radius: only Go and Markdown files are rewritten.
	return ext == ".go" || markdown.IsExt(ext)
}
