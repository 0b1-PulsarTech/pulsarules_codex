package clean

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

type Report struct {
	Path       string
	Acted      []mark.Mark
	Remaining  []mark.Mark
	Changed    bool
	SkipReason string
}

func (report Report) Skipped() bool { return report.SkipReason != "" }

type Cleaner struct {
	root string
}

func New(root string) *Cleaner { return &Cleaner{root: root} }

func (cleaner *Cleaner) Inspect(path string) (Report, error) {
	report, src, err := cleaner.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}
	report.Remaining = mark.Scan(src)
	return report, nil
}

func (cleaner *Cleaner) CleanFile(path string) (Report, error) {
	report, src, err := cleaner.read(path)
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
	if err = cleaner.verify(report.Path); err != nil {
		return report, err
	}
	report.Acted, report.Changed = acted, true
	report.Remaining = mark.Scan(cleaned)
	return report, nil
}

func (cleaner *Cleaner) verify(path string) error {
	written, err := os.ReadFile(path) //nolint:gosec // path already passed ResolveInside.
	if err != nil {
		return fmt.Errorf("verify %q: %w", path, err)
	}
	// Trust the write only after the bytes on disk are re-read.
	for _, found := range mark.Scan(string(written)) {
		if found.Class == mark.ClassStrip || found.Class == mark.ClassSpace {
			return fmt.Errorf("verify %q: %s survived the write", path, found.Name)
		}
	}
	return nil
}
