package clean

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/langspec"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

var errWriteMismatch = errors.New("the bytes on disk differ from the cleaned text")

type Report struct {
	Path       string
	Acted      []mark.Mark
	Remaining  []mark.Mark
	Held       []mark.Mark
	Changed    bool
	SkipReason string
	literals   langspec.Literals
}

func (report Report) Skipped() bool { return report.SkipReason != "" }

func (report Report) SkipNote() string {
	note := "skipped (" + report.SkipReason + ")"
	if len(report.Remaining) == 0 {
		return note
	}
	return note + ": " + markerLines(report.Remaining)
}

func (report Report) IsHeld(found mark.Mark) bool {
	_, held := slices.BinarySearchFunc(report.Held, found.Offset, func(
		each mark.Mark, offset int,
	) int {
		return each.Offset - offset
	})
	return held
}

func (report Report) Removable() int {
	count := 0
	for _, found := range report.Remaining {
		if found.Class == mark.ClassStrip || found.Class == mark.ClassSpace {
			count++
		}
	}
	return count - len(report.Held)
}

func lexGate(report Report, src string) (Report, langspec.Literals) {
	literals := langspec.ScanLiterals(filepath.Ext(report.Path), src)
	if literals.Incomplete() {
		// With no trustworthy literal boundaries, any removal could eat a string's data.
		report.SkipReason = "does not lex: unterminated comment or literal"
		report.Remaining = mark.Scan(src)
	}
	return report, literals
}

type Cleaner struct {
	root string
}

func New(root string) *Cleaner { return &Cleaner{root: root} }

func (cleaner *Cleaner) Inspect(path string) (Report, error) {
	report, src, err := cleaner.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}
	report, literals := lexGate(report, src)
	if !report.Skipped() {
		report = settle(report, src, literals)
	}
	return report, nil
}

func (cleaner *Cleaner) CleanFile(path string) (Report, error) {
	report, src, err := cleaner.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}

	report, literals := lexGate(report, src)
	if report.Skipped() {
		return report, nil
	}
	cleaned, acted := mark.Clean(src, protection(report.Path, src, literals))
	if len(acted) == 0 {
		return settle(report, src, literals), nil
	}

	stat, err := os.Stat(report.Path)
	if err != nil {
		return report, fmt.Errorf("stat %q: %w", path, err)
	}
	if err = fsx.WriteFileAtomic(report.Path, []byte(cleaned), stat.Mode().Perm()); err != nil {
		return report, err
	}
	if err = verify(report.Path, cleaned); err != nil {
		return report, err
	}
	report.Acted, report.Changed = acted, true
	return rescan(report, cleaned), nil
}

func verify(path, want string) error {
	written, err := os.ReadFile(path) //nolint:gosec // path already passed ResolveInside.
	if err != nil {
		return fmt.Errorf("verify %q: %w", path, err)
	}
	// Trust the write only after the bytes on disk are re-read and equal what was written.
	if !bytes.Equal(written, []byte(want)) {
		return fmt.Errorf("verify %q: %w", path, errWriteMismatch)
	}
	return nil
}
