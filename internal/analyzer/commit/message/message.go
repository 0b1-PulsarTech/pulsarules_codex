package message

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
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

func (a *Analyzer) ID() string { return "commit-lint" }

func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if ctx.CommitMsg == "" {
		return nil
	}

	msg := commitmsg.Parse(ctx.CommitMsg)
	windowCfg := a.windowCfg.merge(ctx.Params(a.ID()))

	findings := make([]pass.Finding, 0, typicalFindingCapacity)
	findings = append(findings, Validate(msg, a.ruleCfg, a.ruleReporters)...)
	findings = append(findings, emojiCheck{
		Message:   msg,
		Catalog:   a.catalog,
		History:   historySubjects(ctx.GitHistory),
		Config:    windowCfg,
		Reporters: a.emojiReporters,
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
	cfg.HardWindow = params.Int("emoji_hard_window", cfg.HardWindow)
	cfg.SoftWindow = params.Int("emoji_soft_window", cfg.SoftWindow)
	cfg.Suggestions = params.Int("emoji_suggestions", cfg.Suggestions)
	return cfg
}
