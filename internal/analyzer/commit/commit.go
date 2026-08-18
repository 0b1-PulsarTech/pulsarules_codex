package commit

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

var _ core.Analyzer = (*Analyzer)(nil)

// Analyzer implements core.Analyzer. It parses a commit message from
// the pipeline context, validates it against all commit rules (format, emoji
// catalog, repetition window, tool-attribution trailers), and returns
// findings.
type Analyzer struct {
	catalog   *emoji.Catalog
	ruleCfg   RuleConfig
	windowCfg EmojiWindowConfig
}

// NewAnalyzer creates an Analyzer with default rule and window
// configs, validating against the given emoji catalog.
func NewAnalyzer(catalog *emoji.Catalog) *Analyzer {
	return &Analyzer{
		catalog:   catalog,
		ruleCfg:   DefaultRuleConfig(),
		windowCfg: DefaultEmojiWindowConfig(),
	}
}

func (a *Analyzer) ID() string { return "commit-lint" }

func (a *Analyzer) Name() string { return "Commit lint" }

func (a *Analyzer) Description() string {
	return "Validates commit message format, emoji catalog, repetition window, and trailer rules"
}

func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Category() core.Category { return core.CatCommit }

// Needs declares what the analyzer requires from the pipeline context. Git
// history is deliberately NOT required: the pipeline skips an analyzer whose
// requirements are unmet, and demanding history would disable format and
// catalog checks entirely on a repository's first commit.
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

// Analyze parses the commit message from ctx.CommitMsg, validates it, and
// returns findings.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.CommitMsg == "" {
		return nil
	}

	msg := commitmsg.Parse(ctx.CommitMsg)
	windowCfg := a.windowCfg.merge(ctx.Params(a.ID()))

	findings := make([]core.Finding, 0, typicalFindingCapacity)
	findings = append(findings, Validate(msg, a.ruleCfg)...)
	findings = append(findings, EmojiCheck{
		Message: msg,
		Catalog: a.catalog,
		History: historySubjects(ctx.GitHistory),
		Config:  windowCfg,
	}.ValidateEmoji()...)

	return findings
}

func historySubjects(entries []core.GitCommitEntry) []string {
	subjects := make([]string, len(entries))
	for index, entry := range entries {
		subjects[index] = entry.Subject
	}
	return subjects
}

// merge overlays the runtime analyzer params onto the compiled-in defaults so
// a project can widen or narrow the windows without a rebuild.
func (cfg EmojiWindowConfig) merge(params core.ParamSet) EmojiWindowConfig {
	cfg.HardWindow = params.Int("emoji_hard_window", cfg.HardWindow)
	cfg.SoftWindow = params.Int("emoji_soft_window", cfg.SoftWindow)
	cfg.Suggestions = params.Int("emoji_suggestions", cfg.Suggestions)
	return cfg
}
