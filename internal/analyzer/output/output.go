package output

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var _ core.InPlaceAnalyzer = (*Analyzer)(nil)

// Analyzer runs at StageOutput. It sorts the accumulated findings by
// severity (error first, then warning, then info), then by file and line, and
// removes exact duplicates (same AnalyzerID, File, Line, Message). It mutates
// ctx.Findings in place and returns nil.
type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (a *Analyzer) ID() string { return "output" }

func (a *Analyzer) Stage() core.StageID { return core.StageOutput }

// TransformsInPlace marks Analyzer as a core.InPlaceAnalyzer: it sorts and
// deduplicates ctx.Findings in place rather than contributing new findings.
func (a *Analyzer) TransformsInPlace() {}

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
