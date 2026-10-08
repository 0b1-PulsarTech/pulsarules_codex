package governance

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type Result struct {
	Findings            []finding.Finding
	SuppressedGenerated int
}

func splitGenerated(ctx *pass.Context, findings []finding.Finding) Result {
	known := make(map[string]bool, len(ctx.ChangedFiles))
	for _, fc := range ctx.ChangedFiles {
		known[fc.Path] = fc.Generated
	}

	kept := make([]finding.Finding, 0, len(findings))
	suppressed := 0
	for _, finding := range findings {
		// A finding with no file (a commit-message rule) belongs to no file and is never
		// suppressed; a generated file's are, because the next codegen run undoes any edit.
		if finding.File != "" && isGeneratedFile(finding.File, known, ctx.Sources) {
			suppressed++
			continue
		}
		kept = append(kept, finding)
	}
	return Result{Findings: kept, SuppressedGenerated: suppressed}
}

func isGeneratedFile(path string, known map[string]bool, provider source.Provider) bool {
	if generated, ok := known[path]; ok {
		return generated
	}
	generated := hasAtlasSum(path, provider)
	if provider != nil && !generated {
		if src, found := provider.Read(path); found {
			generated = lexer.IsGeneratedSource(src)
		}
	}
	known[path] = generated
	return generated
}

func memoAtlas(path string, atlasDirs map[string]bool, provider source.Provider) bool {
	if !strings.EqualFold(filepath.Ext(path), ".sql") {
		return false
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	generated, ok := atlasDirs[dir]
	if !ok {
		generated = hasAtlasSum(path, provider)
		atlasDirs[dir] = generated
	}
	return generated
}

func hasAtlasSum(path string, provider source.Provider) bool {
	if provider == nil || !strings.EqualFold(filepath.Ext(path), ".sql") {
		return false
	}
	_, found := provider.Read(filepath.ToSlash(filepath.Join(filepath.Dir(path), "atlas.sum")))
	return found
}
