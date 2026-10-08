package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func (analyzer *Analyzer) checkEntry(
	root string, entry core.DelegatedCommand,
) (kind format, dir, fault string) {
	var known bool
	kind, known = formats()[entry.Format]
	switch {
	case len(entry.Run) == 0:
		return kind, "", "run is empty"
	case !known:
		return kind, "", fmt.Sprintf("unknown format %q", entry.Format)
	}
	dir, fault = dirFault(root, entry.Dir)
	switch {
	case fault != "":
		return kind, "", fault
	case entry.Rule != "" && analyzer.index != nil && !analyzer.index.RuleExists(entry.Rule):
		return kind, "", fmt.Sprintf("rule %q is not in the knowledge index", entry.Rule)
	}
	return kind, dir, ""
}

func dirFault(root, name string) (dir, fault string) {
	if name == "" {
		return root, ""
	}
	if !filepath.IsLocal(name) {
		return "", fmt.Sprintf("dir %q must stay inside the project", name)
	}
	var err error
	dir, err = filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil || !isDir(dir) {
		return "", fmt.Sprintf("dir %q is not a directory", name)
	}
	if rel, relErr := filepath.Rel(root, dir); relErr != nil || !filepath.IsLocal(rel) {
		return "", fmt.Sprintf("dir %q resolves outside the project", name)
	}
	return dir, ""
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
