package commit

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

// AnalyzerID is the stable ID this analyzer reports findings under, and the
// key its runtime params are looked up by - exported so the config
// projection in internal/analysis.withEmojiParams spells it once.
const AnalyzerID = "commit-lint"

// Param keys the pipeline projects config.EmojiConfig's fields onto (see
// internal/analysis.withEmojiParams), read back by merge below - the one
// place each name is spelled, instead of a string literal on both ends of
// the config-to-param hop.
const (
	ParamEmojiHardWindow  = "emoji_hard_window"
	ParamEmojiSoftWindow  = "emoji_soft_window"
	ParamEmojiSuggestions = "emoji_suggestions"
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

func (a *Analyzer) ID() string { return AnalyzerID }

func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

// Analyze parses the commit message from ctx.CommitMsg, validates it, and
// returns findings.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.CommitMsg == "" {
		return nil
	}

	msg := commitmsg.Parse(ctx.CommitMsg)
	windowCfg := a.windowCfg.merge(ctx.Params(a.ID()))

	findings := make([]core.Finding, 0, typicalFindingCapacity)
	findings = append(findings, Validate(msg, a.ruleCfg, defaultRuleReporters().resolved(ctx))...)
	findings = append(findings, EmojiCheck{
		Message:   msg,
		Catalog:   a.catalog,
		History:   historySubjects(ctx.GitHistory),
		Config:    windowCfg,
		Reporters: defaultEmojiReporters().resolved(ctx),
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
	cfg.HardWindow = params.Int(ParamEmojiHardWindow, cfg.HardWindow)
	cfg.SoftWindow = params.Int(ParamEmojiSoftWindow, cfg.SoftWindow)
	cfg.Suggestions = params.Int(ParamEmojiSuggestions, cfg.Suggestions)
	return cfg
}
