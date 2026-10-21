package pass

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

type (
	Descriptor struct {
		ID     string
		Rule   string
		Params []string
	}

	Analyzer interface {
		Analyze(ctx *Context) []finding.Finding
	}
)
