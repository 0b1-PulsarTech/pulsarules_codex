package commit

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

const AnalyzerID = "commit-lint"

const (
	ParamEmojiHardWindow  = "emoji_hard_window"
	ParamEmojiSoftWindow  = "emoji_soft_window"
	ParamEmojiSuggestions = "emoji_suggestions"
)

var _ core.Analyzer = (*Analyzer)(nil)

type Analyzer struct {
	catalog   *emoji.Catalog
	ruleCfg   RuleConfig
	windowCfg EmojiWindowConfig

	ruleReporters  ruleReporters
	emojiReporters emojiReporters
}

func NewAnalyzer(catalog *emoji.Catalog) *Analyzer {
	return &Analyzer{
		catalog:   catalog,
		ruleCfg:   DefaultRuleConfig(),
		windowCfg: DefaultEmojiWindowConfig(),

		ruleReporters:  defaultRuleReporters(),
		emojiReporters: defaultEmojiReporters(),
	}
}

func (a *Analyzer) ID() string { return AnalyzerID }

func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.CommitMsg == "" {
		return nil
	}

	msg := commitmsg.Parse(ctx.CommitMsg)
	windowCfg := a.windowCfg.merge(ctx.Params(a.ID()))

	findings := make([]core.Finding, 0, typicalFindingCapacity)
	findings = append(findings, Validate(msg, a.ruleCfg, a.ruleReporters.resolved(ctx))...)
	findings = append(findings, EmojiCheck{
		Message:   msg,
		Catalog:   a.catalog,
		History:   historySubjects(ctx.GitHistory),
		Config:    windowCfg,
		Reporters: a.emojiReporters.resolved(ctx),
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

func (cfg EmojiWindowConfig) merge(params core.ParamSet) EmojiWindowConfig {
	cfg.HardWindow = params.Int(ParamEmojiHardWindow, cfg.HardWindow)
	cfg.SoftWindow = params.Int(ParamEmojiSoftWindow, cfg.SoftWindow)
	cfg.Suggestions = params.Int(ParamEmojiSuggestions, cfg.Suggestions)
	return cfg
}
