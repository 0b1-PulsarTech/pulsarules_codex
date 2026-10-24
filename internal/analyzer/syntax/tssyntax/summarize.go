package tssyntax

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/sandbox"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/facts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/tsparse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/budget"
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

func handleRequest(request sandbox.Request) *syntax.File {
	source := request.Source
	summary := newSummary(source.Path)
	options := facts.Options{IsTest: source.IsTest, Local: request.Local}
	ext := strings.ToLower(filepath.Ext(source.Path))
	return guarded(summary, func() { dispatch(summary, options, ext, string(source.Content)) })
}

func dispatch(summary *syntax.File, options facts.Options, ext, text string) {
	if summary.Language == language.Svelte {
		summarizeSvelte(summary, options, text)
		return
	}
	kind := scriptKind(summary.Language, ext)
	summarizeScript(summary, options, scriptBlock{text: text, kind: kind})
}

func scriptKind(id language.ID, ext string) core.ScriptKind {
	jsx := ext == ".tsx" || ext == ".jsx"
	switch {
	case id == language.TypeScript && jsx:
		return core.ScriptKindTSX
	case id == language.TypeScript:
		return core.ScriptKindTS
	case jsx:
		return core.ScriptKindJSX
	}
	return core.ScriptKindJS
}

type scriptBlock struct {
	text  string
	kind  core.ScriptKind
	index int
}

func summarizeScript(summary *syntax.File, options facts.Options, script scriptBlock) {
	text := script.text
	if reason := budget.Check(text); reason != "" {
		summary.ParseError = reason
		return
	}
	root := tsparse.Parse(text, script.kind)
	part, err := facts.Collect(root, text, options)
	if err != nil {
		summary.ParseError = fmt.Sprintf("skipped: %v", err)
		return
	}
	merge(summary, part, script.index)
	summary.ParseError = tsparse.FirstError(root, text)
}

func merge(target *syntax.File, part syntax.File, block int) {
	target.Functions = append(target.Functions, part.Functions...)
	target.Blocks = append(target.Blocks, part.Blocks...)
	target.Names = append(target.Names, part.Names...)
	target.DocSites = append(target.DocSites, part.DocSites...)
	target.Elses = append(target.Elses, part.Elses...)
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
