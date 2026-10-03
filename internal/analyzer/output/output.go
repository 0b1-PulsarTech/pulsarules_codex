package output

import (
	"cmp"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

var _ pass.InPlaceAnalyzer = (*Analyzer)(nil)

type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (analyzer *Analyzer) ID() string { return "output" }

func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageOutput }

func (analyzer *Analyzer) TransformsInPlace() {}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
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
