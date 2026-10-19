package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type entryTarget struct {
	kind format
	dir  string
}

func (analyzer *Analyzer) checkEntry(
	root string, entry pass.DelegatedCommand,
) (entryTarget, error) {
	if len(entry.Run) == 0 {
		return entryTarget{}, errors.New("run is empty")
	}
	kind, known := formats()[entry.Format]
	if !known {
		return entryTarget{}, fmt.Errorf("unknown format %q", entry.Format)
	}
	dir, err := resolveDir(root, entry.Dir)
	if err != nil {
		return entryTarget{}, err
	}
	if entry.Rule != "" && analyzer.index != nil && !analyzer.index.RuleExists(entry.Rule) {
		return entryTarget{}, fmt.Errorf("rule %q is not in the knowledge index", entry.Rule)
	}
	return entryTarget{kind: kind, dir: dir}, nil
}

func resolveDir(root, name string) (string, error) {
	if name == "" {
		return root, nil
	}
	if !filepath.IsLocal(name) {
		return "", fmt.Errorf("dir %q must stay inside the project", name)
	}
	dir, err := filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil || !isDir(dir) {
		return "", fmt.Errorf("dir %q is not a directory", name)
	}
	if rel, relErr := filepath.Rel(root, dir); relErr != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("dir %q resolves outside the project", name)
	}
	return dir, nil
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func resolveInDir(dir, name string) string {
	if filepath.Base(name) != name && !filepath.IsAbs(name) {
		return filepath.Join(dir, name)
	}
	return name
}
