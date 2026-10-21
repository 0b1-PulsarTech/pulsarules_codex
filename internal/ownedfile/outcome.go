package ownedfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

type Outcome int

const (
	Kept Outcome = iota
	written
	Removed
)

func WriteOwned(
	path string, body []byte, mode fs.FileMode, makeDir func(dir string) error,
) (Outcome, error) {
	// makeDir lets each host phrase a mkdir failure: their error texts predate this writer.
	exists, ours, err := Check(path)
	if err != nil {
		return Kept, fmt.Errorf("check %q: %w", path, err)
	}
	if exists && !ours {
		return Kept, nil
	}
	if err = makeDir(filepath.Dir(path)); err != nil {
		return Kept, err
	}
	if err = os.WriteFile(path, body, mode); err != nil {
		return Kept, fmt.Errorf("write %q: %w", path, err)
	}
	return written, nil
}

func RemoveOwned(path string) (Outcome, error) {
	// An absent file is not ours: a caller that listed the directory first can lose a race.
	_, ours, err := Check(path)
	if err != nil {
		return Kept, fmt.Errorf("check %q: %w", path, err)
	}
	if !ours {
		return Kept, nil
	}
	if err = os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Kept, nil
		}
		return Kept, fmt.Errorf("remove %q: %w", path, err)
	}
	return Removed, nil
}

func (outcome Outcome) ReportWrite(path, wroteFormat string) report.Report {
	var rpt report.Report
	if outcome == written {
		rpt.Note(wroteFormat, path)
	} else {
		rpt.Warn("kept existing user-authored %s (not overwritten)", path)
	}
	return rpt
}
