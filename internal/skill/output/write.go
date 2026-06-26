package output

import (
	"fmt"
	"os"
	"path"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// writeFile creates parent directories and writes content with gosec-safe perms.
func writeFile(filePath, content string) error {
	if err := os.MkdirAll(path.Dir(filePath), fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(content), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", filePath, err)
	}
	return nil
}

// WriteDoc writes body to <dir>/<docName> plus a sibling .gitignore that
// ignores both docName and itself, so generated skill/workflow output is
// untracked by default; delete that .gitignore to commit the doc to a branch.
func WriteDoc(dir, docName, body string) error {
	if err := writeFile(path.Join(dir, docName), body); err != nil {
		return fmt.Errorf("write %q: %w", docName, err)
	}
	gitignoreBody := docName + "\n.gitignore\n"
	if err := writeFile(path.Join(dir, ".gitignore"), gitignoreBody); err != nil {
		return fmt.Errorf("gitignore for %q: %w", docName, err)
	}
	return nil
}
