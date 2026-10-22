package cleaner

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

var errWriteMismatch = errors.New("the bytes on disk differ from the cleaned text")

type Report struct {
	Path       string
	Acted      []invisible.Mark
	Remaining  []invisible.Mark
	Held       []invisible.Mark
	Changed    bool
	SkipReason string
	literals   language.Literals
}

func (report Report) Skipped() bool { return report.SkipReason != "" }

func (report Report) SkipNote() string {
	note := "skipped (" + report.SkipReason + ")"
	if len(report.Remaining) == 0 {
		return note
	}
	return note + ": " + markerLines(report.Remaining)
}

func (report Report) IsHeld(found invisible.Mark) bool {
	_, held := slices.BinarySearchFunc(report.Held, found.Offset, func(
		each invisible.Mark, offset int,
	) int {
		return each.Offset - offset
	})
	return held
}

func (report Report) Removable() int {
	count := 0
	for _, found := range report.Remaining {
		if found.Class.Removable() {
			count++
		}
	}
	return count - len(report.Held)
}

func lexGate(report Report, src string) (Report, language.Literals) {
	literals := language.ScanLiterals(filepath.Ext(report.Path), src)
	if literals.Incomplete() {
		// With no trustworthy literal boundaries, any removal could eat a string's data.
		report.SkipReason = "does not lex: unterminated comment or literal"
		report.Remaining = invisible.Scan(src)
	}
	return report, literals
}

type Cleaner struct {
	root string
}

func New(root string) *Cleaner { return &Cleaner{root: root} }

func (scrub *Cleaner) Inspect(path string) (Report, error) {
	report, src, err := scrub.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}
	report, literals := lexGate(report, src)
	if !report.Skipped() {
		report = settle(report, src, literals)
	}
	return report, nil
}

func (scrub *Cleaner) CleanFile(path string) (Report, error) {
	report, src, err := scrub.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}

	report, literals := lexGate(report, src)
	if report.Skipped() {
		return report, nil
	}
	cleaned, acted := invisible.Clean(src, protection(report.Path, src, literals))
	if len(acted) == 0 {
		return settle(report, src, literals), nil
	}

	stat, err := os.Stat(report.Path)
	if err != nil {
		return report, fmt.Errorf("stat %q: %w", path, err)
	}
	if err = disk.WriteFileAtomic(report.Path, []byte(cleaned), stat.Mode().Perm()); err != nil {
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
