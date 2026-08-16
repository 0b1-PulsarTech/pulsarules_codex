package cleaner

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
)

type Report struct {
	Path       string
	Acted      []invisible.Mark
	Remaining  []invisible.Mark
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
	report.Remaining = invisible.Scan(src)
	return report, nil
}

func (cleaner *Cleaner) CleanFile(path string) (Report, error) {
	report, src, err := cleaner.read(path)
	if err != nil || report.Skipped() {
		return report, err
	}

	cleaned, acted := invisible.Clean(src)
	if len(acted) == 0 {
		report.Remaining = invisible.Scan(src)
		return report, nil
	}

	stat, err := os.Stat(report.Path)
	if err != nil {
		return report, fmt.Errorf("stat %q: %w", path, err)
	}
	if err = disk.WriteFileAtomic(report.Path, []byte(cleaned), stat.Mode().Perm()); err != nil {
		return report, err
	}
	if err = cleaner.verify(report.Path); err != nil {
		return report, err
	}
	report.Acted, report.Changed = acted, true
	report.Remaining = invisible.Scan(cleaned)
	return report, nil
}

func (cleaner *Cleaner) verify(path string) error {
	written, err := os.ReadFile(path) //nolint:gosec // path already passed ResolveInside.
	if err != nil {
		return fmt.Errorf("verify %q: %w", path, err)
	}
	// Trust the write only after the bytes on disk are re-read.
	for _, found := range invisible.Scan(string(written)) {
		if found.Class == invisible.ClassStrip || found.Class == invisible.ClassSpace {
			return fmt.Errorf("verify %q: %s survived the write", path, found.Name)
		}
	}
	return nil
}
