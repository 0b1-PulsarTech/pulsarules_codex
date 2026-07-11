package delegation

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/gopls"
)

// GoplsAnalyzer delegates Go diagnostics to the gopls language server.
type GoplsAnalyzer struct {
	runner *gopls.Runner
}

// NewGoplsAnalyzer creates an analyzer that delegates to gopls for diagnostics.
func NewGoplsAnalyzer() *GoplsAnalyzer {
	return &GoplsAnalyzer{runner: gopls.NewRunner()}
}

func (a *GoplsAnalyzer) ID() string              { return "gopls" }
func (a *GoplsAnalyzer) Name() string            { return "gopls" }
func (a *GoplsAnalyzer) Description() string     { return "Delegates to gopls for Go diagnostics" }
func (a *GoplsAnalyzer) Stage() core.StageID     { return core.StageStatic }
func (a *GoplsAnalyzer) Category() core.Category { return core.CatSyntax }
func (a *GoplsAnalyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *GoplsAnalyzer) Analyze(_ *core.AnalysisContext) []core.Finding {
	return a.runner.Run()
}
