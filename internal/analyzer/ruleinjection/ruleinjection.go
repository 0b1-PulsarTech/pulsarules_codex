package ruleinjection

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func defaultRuleMap() map[string]string {
	// A new analyzer goes here by exact ID; sub-ID tools also need a lookupRule prefix.
	return map[string]string{
		"file-size":                "effective-go",
		"no-em-dash":               "effective-go",
		"import-groups":            "imports",
		"naming":                   "naming",
		"top-of-file":              "effective-go",
		"big-comment":              "effective-go",
		"commit-lint":              "commits",
		"commit-initial":           "commits",
		"commit-merge":             "commits",
		"commit-emoji-required":    "commits",
		"commit-emoji-count":       "commits",
		"commit-emoji-catalog":     "commits",
		"commit-emoji-prohibited":  "commits",
		"commit-emoji-repeat":      "commits",
		"commit-emoji-soft-repeat": "commits",
		"commit-type-required":     "commits",
		"commit-type-enum":         "commits",
		"commit-scope-charset":     "commits",
		"commit-desc-required":     "commits",
		"commit-desc-length":       "commits",
		"commit-desc-capitalize":   "commits",
		"commit-desc-no-period":    "commits",
		"commit-body-length":       "commits",
		"commit-body-total-length": "commits",
		"commit-no-coauthor":       "commits",
		"commit-move-purity":       "commits",
		"control-flow":             "code-smells",
		"shadowing":                "effective-go",
		"complexity":               "code-smells",
		"arch-boundary":            "dependency-rule",
		"import-cycle":             "module-boundaries",
		"golangci-lint":            "code-smells",
	}
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
			findings[idx].RuleID = lookupRule(rules, findings[idx].AnalyzerID)
		}
		if findings[idx].RuleID != "" {
			findings[idx].RuleBody = index.Body("rules", findings[idx].RuleID)
		}
	}
}

func lookupRule(rules map[string]string, analyzerID string) string {
	if rid, ok := rules[analyzerID]; ok {
		return rid
	}
	// Sub-IDs like "golangci-lint/gocyclo" come from the delegated linter.
	if strings.HasPrefix(analyzerID, "golangci-lint/") {
		return "code-smells"
	}
	return ""
}
