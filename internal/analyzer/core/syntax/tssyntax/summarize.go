package tssyntax

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsfacts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsguard"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsparse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsproc"
)

func guarded(summary *syntax.File, run func()) (result *syntax.File) {
	defer func() {
		if recovered := recover(); recovered != nil {
			summary.ParseError = fmt.Sprintf("skipped: parser failed: %v", recovered)
			result = summary
		}
	}()
	run()
	return summary
}

func handleRequest(request tsproc.Request) *syntax.File {
	summary := newSummary(request.Path)
	options := tsfacts.Options{IsTest: request.IsTest, Local: request.Local}
	ext := strings.ToLower(filepath.Ext(request.Path))
	return guarded(summary, func() { dispatch(summary, options, ext, request.Text) })
}

func dispatch(summary *syntax.File, options tsfacts.Options, ext, text string) {
	if ext == ".svelte" {
		summarizeSvelte(summary, options, text)
		return
	}
	summarizeScript(summary, options, scriptBlock{text: text, kind: scriptKind(ext)})
}

func scriptKind(ext string) core.ScriptKind {
	switch ext {
	case ".ts", ".mts", ".cts":
		return core.ScriptKindTS
	case ".tsx":
		return core.ScriptKindTSX
	case ".jsx":
		return core.ScriptKindJSX
	}
	return core.ScriptKindJS
}

type scriptBlock struct {
	text  string
	kind  core.ScriptKind
	index int
}

func summarizeScript(summary *syntax.File, options tsfacts.Options, script scriptBlock) {
	text := script.text
	if reason := tsguard.Check(text); reason != "" {
		summary.ParseError = reason
		return
	}
	root := tsparse.Parse(text, script.kind)
	facts, err := tsfacts.Collect(root, text, options)
	if err != nil {
		summary.ParseError = fmt.Sprintf("skipped: %v", err)
		return
	}
	merge(summary, facts, script.index)
	summary.ParseError = tsparse.FirstError(root, text)
}

func merge(target *syntax.File, part syntax.File, block int) {
	target.Functions = append(target.Functions, part.Functions...)
	target.Blocks = append(target.Blocks, part.Blocks...)
	target.Names = append(target.Names, part.Names...)
	target.DocSites = append(target.DocSites, part.DocSites...)
	mergeImports(target, part, block)
}

func mergeImports(target *syntax.File, part syntax.File, block int) {
	offset := len(target.ImportGroups)
	for _, imported := range part.Imports {
		imported.Group += offset
		target.Imports = append(target.Imports, imported)
	}
	for _, group := range part.ImportGroups {
		if offset > 0 {
			group = fmt.Sprintf("%s (script %d)", group, block+1)
		}
		target.ImportGroups = append(target.ImportGroups, group)
	}
}
