package message

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

const ID = "commit-lint"

const (
	ParamEmojiHardWindow  = "emoji_hard_window"
	ParamEmojiSoftWindow  = "emoji_soft_window"
	ParamEmojiSuggestions = "emoji_suggestions"
)

var _ pass.Analyzer = (*Analyzer)(nil)

type Analyzer struct {
	catalog   *emoji.Catalog
	ruleCfg   ruleConfig
	windowCfg emojiWindowConfig

	ruleReporters  ruleReporters
	emojiReporters emojiReporters
}

func NewAnalyzer(catalog *emoji.Catalog) *Analyzer {
	return &Analyzer{
		catalog:   catalog,
		ruleCfg:   DefaultRuleConfig(),
		windowCfg: defaultEmojiWindowConfig(),

		ruleReporters:  defaultRuleReporters(),
		emojiReporters: defaultEmojiReporters(),
	}
}

func (analyzer *Analyzer) ID() string { return ID }

func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if ctx.CommitMsg == "" {
		return nil
	}

	msg := commitmsg.Parse(ctx.CommitMsg)
	windowCfg := analyzer.windowCfg.merge(ctx.Params(analyzer.ID()))

	findings := make([]pass.Finding, 0, typicalFindingCapacity)
	findings = append(
		findings,
		Validate(msg, analyzer.ruleCfg, analyzer.ruleReporters.resolved(ctx))...,
	)
	findings = append(findings, emojiCheck{
		Message:   msg,
		Catalog:   analyzer.catalog,
		History:   historySubjects(ctx.GitHistory),
		Config:    windowCfg,
		Reporters: analyzer.emojiReporters.resolved(ctx),
	}.ValidateEmoji()...)

	return findings
}

func historySubjects(entries []pass.GitCommitEntry) []string {
	subjects := make([]string, len(entries))
	for index, entry := range entries {
		subjects[index] = entry.Subject
	}
	return subjects
}

func (cfg emojiWindowConfig) merge(params pass.ParamSet) emojiWindowConfig {
	cfg.HardWindow = params.Int(ParamEmojiHardWindow, cfg.HardWindow)
	cfg.SoftWindow = params.Int(ParamEmojiSoftWindow, cfg.SoftWindow)
	cfg.Suggestions = params.Int(ParamEmojiSuggestions, cfg.Suggestions)
	return cfg
}
