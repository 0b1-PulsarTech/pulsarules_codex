package message

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

const (
	ID         = "commit-lint"
	commitRule = "commits"
)

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

	reporters ruleReporters
}

func NewAnalyzer(catalog *emoji.Catalog) *Analyzer {
	return &Analyzer{
		catalog:   catalog,
		ruleCfg:   defaultRuleConfig(),
		windowCfg: defaultEmojiWindowConfig(),

		reporters: defaultRuleReporters(),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.CommitMsg == "" {
		return nil
	}

	msg := commitmsg.Parse(ctx.CommitMsg)
	windowCfg := analyzer.windowCfg.merge(ctx.Params(ID))

	reporters := analyzer.reporters
	findings := make([]finding.Finding, 0, typicalFindingCapacity)
	findings = append(findings, validate(msg, analyzer.ruleCfg, reporters)...)
	findings = append(findings, emojiCheck{
		message:   msg,
		catalog:   analyzer.catalog,
		history:   historySubjects(ctx.GitHistory),
		config:    windowCfg,
		reporters: reporters,
	}.validate()...)
	for index := range findings {
		findings[index].RuleID = commitRule
	}

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

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   commitRule,
		Params: []string{ParamEmojiHardWindow, ParamEmojiSoftWindow, ParamEmojiSuggestions},
	}
}
