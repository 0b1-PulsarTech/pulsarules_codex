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

const markerLine = "# " + marker.Installed

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
	missing := missingEntries(lines, entries)
	if len(missing) == 0 {
		return nil
	}
	content := appendEntries(string(existing), lines, missing)
	//nolint:gosec // path is under the caller's dir.
	if err = os.WriteFile(path, []byte(content), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

func missingEntries(lines, entries []string) []string {
	var missing []string
	for _, entry := range entries {
		if !slices.Contains(lines, entry) {
			missing = append(missing, entry)
		}
	}
	return missing
}

func appendEntries(content string, lines, missing []string) string {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	// The marker is stamped only when this writes, never claiming ownership it did not add.
	if !slices.Contains(lines, markerLine) {
		content += markerLine + "\n"
	}
	for _, entry := range missing {
		content += entry + "\n"
	}
	return content
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

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
	// Only a file carrying the marker is touched, so a hand-authored twin survives.
	if !strings.Contains(string(existing), marker.Installed) {
		return nil, nil
	}

	lines := splitLines(string(existing))
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if slices.Contains(entries, line) {
			// Only matched lines are returned, so an empty result means the file is untouched.
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
