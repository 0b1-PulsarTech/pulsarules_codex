package filecount

import (
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type anchor struct {
	dir  string
	file string
}

func changedAnchors(
	ctx *pass.Context,
	eligible func(pass.FileChange) bool,
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
		case fc.Generated || lexer.IsGeneratedSource(src):
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
