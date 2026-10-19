package command

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type listing map[string]struct{}

func listingOf(provider source.Provider) listing {
	if provider == nil {
		return nil
	}
	files := listing{}
	provider.Walk(func(path, _ string) bool {
		files[filepath.ToSlash(path)] = struct{}{}
		return true
	})
	return files
}

func (files listing) holds(root, dir string) bool {
	rel, err := filepath.Rel(root, dir)
	if files == nil || err != nil {
		return true
	}
	if rel == "." {
		return len(files) > 0
	}
	prefix := filepath.ToSlash(rel) + "/"
	for path := range files {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

type verdict int

const (
	dropped verdict = iota
	excluded
	kept
)

func (files listing) judge(root, dir, rel string) verdict {
	if files == nil {
		return kept
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	if stat, err := os.Stat(full); err != nil || !stat.Mode().IsRegular() {
		return dropped
	}
	if _, listed := files[rel]; !listed && !files.underListedLink(rel) {
		if files.escapesToListed(root, dir, full) {
			return kept
		}
		return excluded
	}
	return kept
}

func (files listing) escapesToListed(root, dir, full string) bool {
	target, err := filepath.EvalSymlinks(full)
	if err != nil {
		return false
	}
	if _, within := RelativeTo(dir, dir, target); within {
		return false
	}
	rel, inside := RelativeTo(root, root, target)
	_, listed := files[rel]
	return inside && listed
}

func (files listing) underListedLink(rel string) bool {
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		if _, listed := files[dir]; listed {
			return true
		}
	}
	return false
}
