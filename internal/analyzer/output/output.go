package output

import (
	"cmp"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (a *Analyzer) ID() string { return "output" }

func (a *Analyzer) Stage() pass.StageID      { return pass.StageOutput }
func (a *Analyzer) Needs() pass.Requirements { return pass.Requirements{} }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if len(ctx.Findings) == 0 {
		return nil
	}
	slices.SortStableFunc(ctx.Findings, func(left, right pass.Finding) int {
		return cmp.Or(
			cmp.Compare(right.Severity, left.Severity),
			strings.Compare(left.File, right.File),
			cmp.Compare(left.Line, right.Line),
		)
	})
	ctx.Findings = slices.CompactFunc(ctx.Findings, func(left, right pass.Finding) bool {
		return left.AnalyzerID == right.AnalyzerID &&
			left.File == right.File &&
			left.Line == right.Line &&
			left.Message == right.Message
	})
	return nil
}
