package pass

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

type Analyzer interface {
	ID() string
	Stage() finding.StageID
	Needs() finding.Requirements
	Analyze(ctx *Context) []finding.Finding
}
