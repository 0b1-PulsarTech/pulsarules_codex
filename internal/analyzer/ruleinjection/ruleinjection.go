package ruleinjection

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func defaultRuleMap() map[string]string {
	// A new analyzer goes here by exact ID.
	return map[string]string{}
}

type Analyzer struct {
	index *knowledge.Index
}

func NewAnalyzer(index *knowledge.Index) *Analyzer {
	// A nil index is accepted and skips injection.
	return &Analyzer{index: index}
}

func (a *Analyzer) ID() string               { return "rule-injection" }
func (a *Analyzer) Stage() core.StageID      { return core.StageRuleInjection }
func (a *Analyzer) Needs() core.Requirements { return core.Requirements{} }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	injectRuleBodies(ctx.Findings, a.index)
	return nil
}

func injectRuleBodies(findings []core.Finding, index *knowledge.Index) {
	if index == nil {
		return
	}
	rules := defaultRuleMap()
	for idx := range findings {
		if findings[idx].RuleBody != "" {
			continue
		}
		if findings[idx].RuleID == "" {
			findings[idx].RuleID = rules[findings[idx].AnalyzerID]
		}
		if findings[idx].RuleID != "" {
			findings[idx].RuleBody = index.Body("rules", findings[idx].RuleID)
		}
	}
}
