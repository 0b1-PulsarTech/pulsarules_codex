package rulesummary

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func defaultRuleMap() map[string]string {
	// A new analyzer goes here by exact ID.
	return map[string]string{
		"file-size":     "effective-go",
		"no-em-dash":    "effective-go",
		"import-groups": "imports",
		"naming":        "naming",
		"top-of-file":   "effective-go",
		"big-comment":   "effective-go",
		"control-flow":  "code-smells",
		"shadowing":     "effective-go",
		"complexity":    "code-smells",
		"arch-boundary": "dependency-rule",
		"import-cycle":  "module-boundaries",
	}
}

type Analyzer struct {
	index *knowledge.Index
}

func NewAnalyzer(index *knowledge.Index) *Analyzer {
	// A nil index is accepted and skips injection.
	return &Analyzer{index: index}
}

func (a *Analyzer) ID() string                  { return "rule-injection" }
func (a *Analyzer) Stage() finding.StageID      { return finding.StageRuleInjection }
func (a *Analyzer) Needs() finding.Requirements { return finding.Requirements{} }

func (a *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	injectRuleBodies(ctx.Findings, a.index)
	return nil
}

func injectRuleBodies(findings []finding.Finding, index *knowledge.Index) {
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
