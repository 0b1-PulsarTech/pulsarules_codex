package source

import (
	"path/filepath"
	"slices"
	"strings"
)

func DefaultSkipDirs() []string {
	return []string{
		".git", ".claude", ".codex", ".opencode", ".idea", ".vscode",
		"build", "generated", "testdata", "dist", "out", "coverage",
		"vendor",
		"node_modules", ".next", ".nuxt", ".svelte-kit",
		".dart_tool",
		"__pycache__", ".venv", "venv",
		"target", ".gradle",
		"Pods",
	}
}

func GitSkipDirs() []string {
	// Git ignore already excludes build output; these names only hide real source.
	generic := []string{"build", "generated", "dist", "out", "coverage", "target"}
	return slices.DeleteFunc(DefaultSkipDirs(), func(name string) bool {
		return slices.Contains(generic, name)
	})
}

type SkipSet struct {
	dirs map[string]bool
}

func NewSkipSet(names []string) SkipSet {
	dirs := make(map[string]bool, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			dirs[trimmed] = true
		}
	}
	return SkipSet{dirs: dirs}
}

func (skips SkipSet) orDefault() SkipSet {
	// An empty list is a deliberate choice to skip nothing, not unconfigured.
	if skips.dirs == nil {
		return NewSkipSet(DefaultSkipDirs())
	}
	return skips
}

func (skips SkipSet) SkipsDir(name string) bool { return skips.orDefault().dirs[name] }

func (skips SkipSet) SkipsPath(path string) bool {
	// Only directory segments count: a file named vendor is still analyzed.
	dir := filepath.ToSlash(filepath.Dir(filepath.ToSlash(path)))
	if dir == "." || dir == "/" {
		return false
	}
	resolved := skips.orDefault()
	for segment := range strings.SplitSeq(dir, "/") {
		if resolved.dirs[segment] {
			return true
		}
	}
	return false
}
