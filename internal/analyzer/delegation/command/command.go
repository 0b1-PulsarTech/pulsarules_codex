package command

import (
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

const (
	AnalyzerID = "delegated-command"

	commandTimeout = 5 * time.Minute

	maxOutputBytes = 32 << 20
)

type Analyzer struct {
	index *knowledge.Index
}

func NewAnalyzer(index *knowledge.Index) *Analyzer {
	return &Analyzer{index: index}
}

func (analyzer *Analyzer) ID() string          { return AnalyzerID }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.Config == nil || len(ctx.Config.Delegated) == 0 {
		return nil
	}
	var findings []core.Finding
	listed := listingOf(ctx.Sources)
	for _, entry := range ctx.Config.Delegated {
		findings = append(findings, analyzer.runEntry(ctx.ProjectDir, listed, entry)...)
	}
	return findings
}
