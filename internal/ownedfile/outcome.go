package ownedfile

import (
	"bytes"
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
) (Outcome, Ledger, error) {
	// makeDir lets each host phrase a mkdir failure: their error texts predate this writer.
	var ledger Ledger
	content, exists, err := readExisting(path)
	if err != nil {
		return Kept, ledger, fmt.Errorf("check %q: %w", path, err)
	}
	if exists && !bytes.Contains(content, []byte(Marker)) {
		return Kept, ledger, nil
	}
	if err = makeDir(filepath.Dir(path)); err != nil {
		return Kept, ledger, err
	}
	if exists && Edited(content) {
		if err = saveEdits(path, &ledger); err != nil {
			return Kept, ledger, err
		}
	}
	if err = os.WriteFile(path, Stamp(body), mode); err != nil {
		return Kept, ledger, fmt.Errorf("write %q: %w", path, err)
	}
	return written, ledger, nil
}

func saveEdits(path string, ledger *Ledger) error {
	backupPath, err := Backup(path)
	if err != nil {
		return err
	}
	ledger.AddBackup(EditedMessage(path, backupPath))
	return nil
}

func RemoveOwned(path string) (Outcome, Ledger, error) {
	// An absent file is not ours: a caller that listed the directory first can lose a race.
	var ledger Ledger
	content, exists, err := readExisting(path)
	if err != nil {
		return Kept, ledger, fmt.Errorf("check %q: %w", path, err)
	}
	if !exists || !bytes.Contains(content, []byte(Marker)) {
		return Kept, ledger, nil
	}
	if Edited(content) {
		// Moved aside rather than deleted, so the edits outlive the uninstall.
		if err = saveEdits(path, &ledger); err != nil {
			return Kept, ledger, err
		}
		return Removed, ledger, nil
	}
	if err = os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Kept, ledger, nil
		}
		return Kept, ledger, fmt.Errorf("remove %q: %w", path, err)
	}
	return Removed, ledger, nil
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
