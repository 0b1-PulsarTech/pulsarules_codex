package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type Result struct {
	Findings            []core.Finding
	SuppressedGenerated int
}

func splitGenerated(ctx *core.AnalysisContext, findings []core.Finding) Result {
	known := make(map[string]bool, len(ctx.ChangedFiles))
	for _, fc := range ctx.ChangedFiles {
		known[fc.Path] = fc.Generated
	}

	kept := make([]core.Finding, 0, len(findings))
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

func isGeneratedFile(path string, known map[string]bool, sources core.SourceProvider) bool {
	if generated, ok := known[path]; ok {
		return generated
	}
	generated := false
	if sources != nil {
		if src, found := sources.Read(path); found {
			generated = lexical.IsGeneratedSource(src)
		}
	}
	known[path] = generated
	return generated
}
