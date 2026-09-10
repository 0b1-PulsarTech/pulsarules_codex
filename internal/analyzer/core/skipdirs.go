package core

import (
	"path/filepath"
	"strings"
)

func DefaultSkipDirs() []string {
	return []string{
		".git", ".claude", ".opencode",
		"build", "generated", "testdata",
		"vendor",
	}
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
