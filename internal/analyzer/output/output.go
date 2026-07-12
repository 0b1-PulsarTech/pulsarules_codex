package output

import (
	"cmp"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (a *Analyzer) ID() string { return "output" }

func (a *Analyzer) Stage() finding.StageID      { return finding.StageOutput }
func (a *Analyzer) Needs() finding.Requirements { return finding.Requirements{} }

func (a *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if len(ctx.Findings) == 0 {
		return nil
	}
	slices.SortStableFunc(ctx.Findings, func(left, right finding.Finding) int {
		return cmp.Or(
			cmp.Compare(right.Severity, left.Severity),
			strings.Compare(left.File, right.File),
			cmp.Compare(left.Line, right.Line),
		)
	})
	ctx.Findings = slices.CompactFunc(ctx.Findings, func(left, right finding.Finding) bool {
		return left.AnalyzerID == right.AnalyzerID &&
			left.File == right.File &&
			left.Line == right.Line &&
			left.Message == right.Message
	})
	return nil
}
