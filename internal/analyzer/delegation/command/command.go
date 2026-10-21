package command

import (
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

const (
	ID = "delegated-command"

	commandTimeout = 5 * time.Minute

	MaxOutputBytes = 32 << 20
)

type Analyzer struct {
	index *knowledge.Index
}

func NewAnalyzer(index *knowledge.Index) *Analyzer {
	return &Analyzer{index: index}
}

func (analyzer *Analyzer) ID() string             { return ID }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.Config == nil || len(ctx.Config.Delegated) == 0 {
		return nil
	}
	var findings []finding.Finding
	listed := listingOf(ctx.Sources)
	for _, entry := range ctx.Config.Delegated {
		findings = append(findings, analyzer.runEntry(ctx.ProjectDir, listed, entry)...)
	}
	return findings
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID}
}
