package output

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer runs at StageOutput. It sorts the accumulated findings by
// severity (error first, then warning, then info), then by file and line, and
// removes exact duplicates (same AnalyzerID, File, Line, Message). It mutates
// ctx.Findings in place and returns nil.
type Analyzer struct{}

// NewOutputAnalyzer creates an output analyzer that sorts and deduplicates findings.
func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (a *Analyzer) ID() string   { return "output" }
func (a *Analyzer) Name() string { return "Output aggregation" }

func (a *Analyzer) Description() string      { return "Sorts, deduplicates, and counts findings" }
func (a *Analyzer) Stage() core.StageID      { return core.StageOutput }
func (a *Analyzer) Category() core.Category  { return core.CatProject }
func (a *Analyzer) Needs() core.Requirements { return core.Requirements{} }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if len(ctx.Findings) == 0 {
		return nil
	}
	slices.SortStableFunc(ctx.Findings, func(a, b core.Finding) int {
		if a.Severity != b.Severity {
			if a.Severity > b.Severity {
				return -1
			}
			return 1
		}
		if a.File != b.File {
			if a.File < b.File {
				return -1
			}
			return 1
		}
		if a.Line < b.Line {
			return -1
		} else if a.Line > b.Line {
			return 1
		}
		return 0
	})
	ctx.Findings = slices.CompactFunc(ctx.Findings, func(a, b core.Finding) bool {
		return a.AnalyzerID == b.AnalyzerID &&
			a.File == b.File &&
			a.Line == b.Line &&
			a.Message == b.Message
	})
	return nil
}
