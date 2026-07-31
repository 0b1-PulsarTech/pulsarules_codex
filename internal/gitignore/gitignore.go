package gitignore

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// Ensure makes sure <dir>/.gitignore contains every entry, appending only the
// ones missing. It creates dir and the .gitignore file if they do not already
// exist, preserves any existing content, and adds a trailing newline first if
// the file did not already end in one.
func Ensure(dir string, entries ...string) error {
	if err := os.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	path := filepath.Join(dir, ".gitignore")
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's dir.
	if err != nil && !os.IsNotExist(err) {
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
