package rulesummary

import (
	"maps"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func defaultRuleMap() map[string]string {
	// A new analyzer goes here by exact ID; sub-ID tools also need a lookupRule prefix.
	rules := map[string]string{
		"file-size":              "effective-go",
		"text-markers":           "effective-go",
		"typographic-markers":    "effective-go",
		"ai-frontmatter":         "effective-go",
		"import-groups":          "imports",
		"naming":                 "naming",
		"top-of-file":            "effective-go",
		"big-comment":            "effective-go",
		"vacuous-doc":            "effective-go",
		"history-narration":      "effective-go",
		"doc-comment":            "effective-go",
		"body-header-comment":    "effective-go",
		"package-level-var":      "code-smells",
		"package-file-count":     "code-placement-go",
		"test-comment":           "effective-go",
		"simplification-path":    "minimalism",
		"single-use-declaration": "minimalism",
		"control-flow":           "code-smells",
		"shadowing":              "effective-go",
		"complexity":             "code-smells",
		// The short-decl-reuse finding comes from the shadowing analyzer, but the fix it asks
		// for, a var declaration above the call, is the named-returns rule's territory.
		"short-decl-reuse":   "named-returns",
		"named-returns":      "named-returns",
		"time-discipline":    "concurrency",
		"context-discipline": "concurrency",
		"pprof-labels":       "security",
		"arch-boundary":      "dependency-rule",
		"import-cycle":       "module-boundaries",
		"golangci-lint":      "code-smells",
	}
	maps.Copy(rules, commitRuleMap())
	return rules
}

func commitRuleMap() map[string]string {
	return map[string]string{
		"branch-name":              "commits",
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
	}
}

var _ pass.InPlaceAnalyzer = (*Analyzer)(nil)

type Analyzer struct {
	index *knowledge.Index
}

func NewAnalyzer(index *knowledge.Index) *Analyzer {
	// A nil index is accepted and skips injection.
	return &Analyzer{index: index}
}

func (a *Analyzer) ID() string          { return "rule-injection" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageRuleInjection }

func (a *Analyzer) TransformsInPlace() {}

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	injectRuleSummaries(ctx.Findings, a.index)
	return nil
}

func injectRuleSummaries(findings []pass.Finding, index *knowledge.Index) {
	if index == nil {
		return
	}
	rules := defaultRuleMap()
	for idx := range findings {
		if findings[idx].RuleSummary != "" {
			continue
		}
		if findings[idx].RuleID == "" {
			findings[idx].RuleID = lookupRule(rules, findings[idx].AnalyzerID)
		}
		if findings[idx].RuleID != "" {
			findings[idx].RuleSummary = index.Summary("rules", findings[idx].RuleID)
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
