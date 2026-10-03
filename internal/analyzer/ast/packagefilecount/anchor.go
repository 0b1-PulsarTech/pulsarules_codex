package packagefilecount

import (
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type anchor struct {
	dir  string
	file string
}

func changedAnchors(
	ctx *core.AnalysisContext,
	eligible func(core.FileChange) bool,
) []anchor {
	var order []string
	seen := make(map[string]bool)
	primary := make(map[string]string)
	fallback := make(map[string]string)
	for _, fc := range ctx.ChangedFiles {
		file := filepath.ToSlash(fc.Path)
		dir := path.Dir(file)
		if !eligible(fc) || isNonPackageDir(dir) {
			continue
		}
		src, ok := ctx.Sources.Read(file)
		if !ok {
			continue
		}
		if !seen[dir] {
			seen[dir] = true
			order = append(order, dir)
		}
		switch {
		case fc.Generated || lexical.IsGeneratedSource(src):
		case isTestFile(file, fc.Extension):
			if fallback[dir] == "" {
				fallback[dir] = file
			}
		case primary[dir] == "":
			primary[dir] = file
		}
	}
	anchors := make([]anchor, 0, len(order))
	for _, dir := range order {
		file := primary[dir]
		if file == "" {
			// A test file is the last resort so a test-only change still surfaces the finding.
			file = fallback[dir]
		}
		if file != "" {
			anchors = append(anchors, anchor{dir: dir, file: file})
		}
	}
	return anchors
}

func malformedAnchor(ctx *core.AnalysisContext, anchors []anchor) string {
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
		if fc.Extension == ".go" && !fc.Generated && !lexical.IsGeneratedSource(src) {
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
