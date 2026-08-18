package clean

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

// maxBytes bounds what the cleaner will read into memory.
//
// why: it runs on every edit, so a checked-in blob must not stall a turn. A
// source file past this size has a bigger problem than a stray marker.
const maxBytes = 1 << 20

// eligible is the mutation blast radius, stated as data rather than as a
// condition buried in a function.
var eligible = map[string]bool{".go": true, ".md": true}

// Report is what one file produced. Acted is empty unless the file was written.
type Report struct {
	Path       string
	Acted      []mark.Mark
	Remaining  []mark.Mark
	Changed    bool
	SkipReason string
}

// Skipped reports whether a precondition refused the file before any read.
func (r Report) Skipped() bool { return r.SkipReason != "" }

// Cleaner removes context-free markers from files under one project root.
type Cleaner struct {
	root string
}

// New returns a Cleaner bound to root; every path it accepts must resolve inside it.
func New(root string) *Cleaner { return &Cleaner{root: root} }

// Inspect reports a file's markers without writing anything.
func (c *Cleaner) Inspect(path string) (Report, error) {
	report, src, err := c.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}
	report.Remaining = mark.Scan(src)
	return report, nil
}

// CleanFile removes the context-free markers and writes the file back, but only
// when there is something to remove. It reports what it acted on and what it
// deliberately left behind.
func (c *Cleaner) CleanFile(path string) (Report, error) {
	report, src, err := c.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}

	cleaned, acted := mark.Clean(src)
	if len(acted) == 0 {
		report.Remaining = mark.Scan(src)
		return report, nil
	}

	stat, err := os.Stat(report.Path)
	if err != nil {
		return report, fmt.Errorf("stat %q: %w", path, err)
	}
	if err = fsx.WriteFileAtomic(report.Path, []byte(cleaned), stat.Mode().Perm()); err != nil {
		return report, err
	}
	if err = c.verify(report.Path); err != nil {
		return report, err
	}
	report.Acted, report.Changed = acted, true
	report.Remaining = mark.Scan(cleaned)
	return report, nil
}

// why: the write is trusted only after the bytes on disk are re-read. A tool
// that reports success and leaves the carrier in place is the failure this
// whole design exists to avoid.
func (c *Cleaner) verify(path string) error {
	written, err := os.ReadFile(path) //nolint:gosec // path already passed ResolveInside.
	if err != nil {
		return fmt.Errorf("verify %q: %w", path, err)
	}
	for _, found := range mark.Scan(string(written)) {
		if found.Class == mark.ClassStrip || found.Class == mark.ClassSpace {
			return fmt.Errorf("verify %q: %s survived the write", path, found.Name)
		}
	}
	return nil
}
