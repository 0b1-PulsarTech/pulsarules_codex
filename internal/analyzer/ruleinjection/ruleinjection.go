package ruleinjection

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// defaultRuleMap maps analyzer IDs to knowledge-base rule IDs for rule-body
// injection at StageRuleInjection. Entries cover both exact analyzer IDs and
// the prefix-matching patterns for delegated tools that inject linter-specific
// sub-IDs. This table is the contract between the pipeline and the knowledge
// base - adding a new analyzer should include a mapping entry.
var defaultRuleMap = map[string]string{
	// static analyzers
	"file-size":     "effective-go",
	"no-em-dash":    "effective-go",
	"import-groups": "imports",
	"naming":        "naming",
	"top-of-file":   "effective-go",
	"big-comment":   "effective-go",
	// commit-lint sub-rules (each has its own AnalyzerID)
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
	// AST analyzers
	"control-flow": "code-smells",
	"shadowing":    "effective-go",
	"complexity":   "code-smells",
	// short-decl-reuse is emitted by the shadowing analyzer but points at the
	// named-returns rule: the fix it asks for is a var declaration above the
	// call, which is that rule's territory, not the shadowing one's.
	"short-decl-reuse": "named-returns",
	"named-returns":    "named-returns",
	// architecture analyzers
	"arch-boundary": "dependency-rule",
	"import-cycle":  "module-boundaries",
	// delegated tool analyzers
	"golangci-lint": "code-smells",
	"gopls":         "effective-go",
}

// Analyzer injects embedded rule markdown bodies into findings
// at StageRuleInjection. It maps each finding's AnalyzerID to a knowledge-base
// rule ID via the defaultRuleMap and looks up the body from the Index.
type Analyzer struct {
	index *knowledge.Index
}

// NewAnalyzer creates a Analyzer that reads rule bodies
// from the given knowledge index. Pass nil to skip injection.
func NewAnalyzer(index *knowledge.Index) *Analyzer {
	return &Analyzer{index: index}
}

func (a *Analyzer) ID() string   { return "rule-injection" }
func (a *Analyzer) Name() string { return "Rule injection" }
func (a *Analyzer) Description() string {
	return "Attaches rule markdown bodies to findings"
}
func (a *Analyzer) Stage() core.StageID      { return core.StageRuleInjection }
func (a *Analyzer) Category() core.Category  { return core.CatCommit }
func (a *Analyzer) Needs() core.Requirements { return core.Requirements{} }

// Analyze delegates to injectRuleBodies, which operates on the full set of
// findings accumulated by earlier pipeline stages.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	injectRuleBodies(ctx.Findings, a.index)
	return nil
}

// injectRuleBodies applies rule-body injection to a finding slice using the
// default rule map and the given knowledge index. It is the same logic that
// Analyzer applies in StageRuleInjection, exposed for use outside
// a full pipeline run. Pass nil index to skip injection.
func injectRuleBodies(findings []core.Finding, index *knowledge.Index) {
	if index == nil {
		return
	}
	for i := range findings {
		if findings[i].RuleBody != "" {
			continue
		}
		if findings[i].RuleID == "" {
			findings[i].RuleID = lookupRule(findings[i].AnalyzerID)
		}
		if findings[i].RuleID != "" {
			findings[i].RuleBody = index.Body("rules", findings[i].RuleID)
		}
	}
}

// lookupRule returns the rule ID for the given analyzer ID, checking exact
// matches first and then prefix patterns for delegated tools.
func lookupRule(analyzerID string) string {
	if rid, ok := defaultRuleMap[analyzerID]; ok {
		return rid
	}
	// golangci-lint produces sub-IDs like "golangci-lint/gocyclo"
	if strings.HasPrefix(analyzerID, "golangci-lint/") {
		return "code-smells"
	}
	return ""
}
