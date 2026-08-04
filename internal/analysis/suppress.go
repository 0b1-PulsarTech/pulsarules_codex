package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Result is what one analysis run produced: the findings that survived
// suppression, plus how many were dropped for living in a generated file.
// The count is carried rather than discarded because a silent suppression is
// worse than an inflated total - a reader has to know the panel is filtered.
type Result struct {
	// Findings are the findings a caller should act on.
	Findings []core.Finding
	// SuppressedGenerated is how many findings fell in generated files.
	SuppressedGenerated int
}

// splitGenerated partitions findings by whether they land in a generated file.
//
// It filters FINDINGS rather than skipping files on the way in, because that
// is the only point every analyzer passes through: golangci-lint and gopls run
// as external processes, and the arch analyzers walk the tree themselves, so
// neither would honour a filter applied to ChangedFiles. It is also what
// yields the exact count the footer reports.
func splitGenerated(ctx *core.AnalysisContext, findings []core.Finding) Result {
	known := make(map[string]bool, len(ctx.ChangedFiles))
	for _, fc := range ctx.ChangedFiles {
		known[fc.Path] = fc.Generated
	}

	kept := make([]core.Finding, 0, len(findings))
	suppressed := 0
	for _, f := range findings {
		// A finding with no file (a commit-message rule) belongs to no file
		// and is never suppressed.
		if f.File != "" && isGeneratedFile(f.File, known, ctx.Sources) {
			suppressed++
			continue
		}
		kept = append(kept, f)
	}
	return Result{Findings: kept, SuppressedGenerated: suppressed}
}

// isGeneratedFile answers for a path the changed set may never have listed: a
// delegated linter reports across the whole module, and the arch analyzers
// walk their own tree. Answers are memoised into known, and the reads behind
// them are already cached by the SourceProvider.
func isGeneratedFile(path string, known map[string]bool, sources core.SourceProvider) bool {
	if generated, ok := known[path]; ok {
		return generated
	}
	generated := false
	if sources != nil {
		if src, found := sources.Read(path); found {
			generated = core.IsGeneratedSource(src)
		}
	}
	known[path] = generated
	return generated
}
