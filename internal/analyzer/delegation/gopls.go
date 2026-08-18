package delegation

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/gopls"
)

// GoplsAnalyzer probes gopls availability on PATH; it does not run gopls
// diagnostics (see gopls.Runner.Run's simplification note for the upgrade
// path to a real diagnostics route).
type GoplsAnalyzer struct {
	runner *gopls.Runner
}

func NewGoplsAnalyzer() *GoplsAnalyzer {
	return &GoplsAnalyzer{runner: gopls.NewRunner()}
}

func (a *GoplsAnalyzer) ID() string   { return "gopls" }
func (a *GoplsAnalyzer) Name() string { return "gopls" }
func (a *GoplsAnalyzer) Description() string {
	return "Probes gopls availability on PATH; reports no diagnostics"
}
func (a *GoplsAnalyzer) Stage() core.StageID     { return core.StageStatic }
func (a *GoplsAnalyzer) Category() core.Category { return core.CatSyntax }
func (a *GoplsAnalyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *GoplsAnalyzer) Analyze(_ *core.AnalysisContext) []core.Finding {
	return a.runner.Run()
}
