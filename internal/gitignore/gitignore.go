package gitignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// markerLine is marker.Installed on its own .gitignore line. A leading "#"
// makes it a plain comment, so writing it never changes what git ignores;
// its presence is the proof Remove needs that a file's managed entries were
// actually written by Ensure and are therefore safe to strip.
const markerLine = "# " + marker.Installed

// Ensure makes sure <dir>/.gitignore contains every entry, appending only
// the missing ones, creating dir/file as needed and preserving existing
// content. It stamps markerLine when it writes new entries - the proof
// Remove later needs to strip them - and leaves an already-complete file
// completely untouched, so Ensure never claims ownership it did not add.
func Ensure(dir string, entries ...string) error {
	if err := os.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	path := filepath.Join(dir, ".gitignore")
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's dir.
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %q: %w", path, err)
	}
	lines := splitLines(string(existing))
	var missing []string
	for _, entry := range entries {
		if !slices.Contains(lines, entry) {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if !slices.Contains(lines, markerLine) {
		content += markerLine + "\n"
	}
	for _, entry := range missing {
		content += entry + "\n"
	}
	//nolint:gosec // path is under the caller's dir.
	if err = os.WriteFile(path, []byte(content), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// Remove strips entries from <dir>/.gitignore, undoing what Ensure added.
// It only touches a file carrying markerLine - proof Ensure wrote it, so a
// hand-authored file with identical entries survives Remove untouched.
// markerLine and the file itself are dropped once nothing is left to
// protect. Idempotent: a missing, foreign, or already-clean file is no error.
func Remove(dir string, entries ...string) (removed []string, err error) {
	path := filepath.Join(dir, ".gitignore")
	var existing []byte
	existing, err = os.ReadFile(path) //nolint:gosec // path is under the caller's dir.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if !strings.Contains(string(existing), marker.Installed) {
		return nil, nil
	}

	lines := splitLines(string(existing))
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if slices.Contains(entries, line) {
			// why: callers report per-file, so an empty removed must stay
			// distinguishable from a real removal - see hook/install's Result.
			removed = append(removed, line)
			continue
		}
		kept = append(kept, line)
	}
	if len(removed) == 0 {
		return nil, nil
	}
	kept = slices.DeleteFunc(kept, func(line string) bool { return line == markerLine })

	if len(kept) == 0 {
		if err = os.Remove(path); err != nil {
			return nil, fmt.Errorf("remove %q: %w", path, err)
		}
		return removed, nil
	}
	content := strings.Join(kept, "\n") + "\n"
	//nolint:gosec // path is under the caller's dir.
	if err = os.WriteFile(path, []byte(content), fsperm.FilePrivate); err != nil {
		return nil, fmt.Errorf("write %q: %w", path, err)
	}
	return removed, nil
}
