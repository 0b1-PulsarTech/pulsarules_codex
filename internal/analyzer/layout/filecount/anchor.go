package filecount

import (
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

type anchor struct {
	dir  string
	file string
}

type dirAnchors struct {
	order    []string
	seen     map[string]bool
	primary  map[string]string
	fallback map[string]string
}

func newDirAnchors() *dirAnchors {
	return &dirAnchors{
		seen:     make(map[string]bool),
		primary:  make(map[string]string),
		fallback: make(map[string]string),
	}
}

func (set *dirAnchors) add(fc pass.FileChange, file string, src []byte) {
	dir := path.Dir(file)
	if !set.seen[dir] {
		set.seen[dir] = true
		set.order = append(set.order, dir)
	}
	switch {
	case fc.Generated || lexer.IsGeneratedSource(src):
	case fc.IsTest:
		if set.fallback[dir] == "" {
			set.fallback[dir] = file
		}
	case set.primary[dir] == "":
		set.primary[dir] = file
	}
}

func (set *dirAnchors) anchors() []anchor {
	found := make([]anchor, 0, len(set.order))
	for _, dir := range set.order {
		file := set.primary[dir]
		if file == "" {
			// A test file is the last resort so a test-only change still surfaces the finding.
			file = set.fallback[dir]
		}
		if file != "" {
			found = append(found, anchor{dir: dir, file: file})
		}
	}
	return found
}

func changedAnchors(
	ctx *pass.Context,
	eligible func(pass.FileChange) bool,
) []anchor {
	set := newDirAnchors()
	for _, fc := range ctx.ChangedFiles {
		file := filepath.ToSlash(fc.Path)
		if !eligible(fc) || gomod.IsPackageDir(path.Dir(file)) {
			continue
		}
		src, ok := ctx.Sources.Read(file)
		if !ok {
			continue
		}
		set.add(fc, file, src)
	}
	return set.anchors()
}

func malformedAnchor(ctx *pass.Context, anchors []anchor) string {
	if len(anchors) > 0 {
		return anchors[0].file
	}
	live := ""
	for _, fc := range ctx.ChangedFiles {
		file := filepath.ToSlash(fc.Path)
		src, ok := ctx.Sources.Read(file)
		if !ok {
			continue
		}
		if fc.Extension == ".go" && !fc.Generated && !lexer.IsGeneratedSource(src) {
			return file
		}
		if live == "" {
			live = file
		}
	}
	if live == "" {
		return filepath.ToSlash(ctx.ChangedFiles[0].Path)
	}
	return live
}
