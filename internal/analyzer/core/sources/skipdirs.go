package sources

import (
	"path/filepath"
	"slices"
	"strings"
)

func DefaultSkipDirs() []string {
	return []string{
		".git", ".claude", ".opencode", ".idea", ".vscode",
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

func (s SkipSet) orDefault() SkipSet {
	// An empty list is a deliberate choice to skip nothing, not unconfigured.
	if s.dirs == nil {
		return NewSkipSet(DefaultSkipDirs())
	}
	return s
}

func (s SkipSet) SkipsDir(name string) bool { return s.orDefault().dirs[name] }

func (s SkipSet) SkipsPath(path string) bool {
	// Only directory segments count: a file named vendor is still analyzed.
	dir := filepath.ToSlash(filepath.Dir(filepath.ToSlash(path)))
	if dir == "." || dir == "/" {
		return false
	}
	resolved := s.orDefault()
	for segment := range strings.SplitSeq(dir, "/") {
		if resolved.dirs[segment] {
			return true
		}
	}
	return false
}
