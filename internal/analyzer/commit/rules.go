package commit

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

type RuleConfig struct {
	MaxSubjectLen      int
	MaxBodyLineLen     int
	MaxBodyLen         int
	RequireEmoji       bool
	RejectToolTrailers bool
}

const typicalFindingCapacity = 8

func DefaultRuleConfig() RuleConfig {
	const (
		defaultMaxSubjectLen  = 72
		defaultMaxBodyLineLen = 100
		defaultMaxBodyLen     = 150
	)
	return RuleConfig{
		MaxSubjectLen:      defaultMaxSubjectLen,
		MaxBodyLineLen:     defaultMaxBodyLineLen,
		MaxBodyLen:         defaultMaxBodyLen,
		RequireEmoji:       true,
		RejectToolTrailers: true,
	}
}

func Validate(msg commitmsg.Message, cfg RuleConfig, reporters ruleReporters) []core.Finding {
	findings := make([]core.Finding, 0, typicalFindingCapacity)

	rawAfterEmoji := strings.TrimSpace(msg.Raw)
	for _, emoji := range msg.Emojis {
		rawAfterEmoji = strings.TrimSpace(strings.TrimPrefix(rawAfterEmoji, ":"+emoji+":"))
	}

	isInitialText := strings.HasPrefix(rawAfterEmoji, "Initial Commit") ||
		strings.HasPrefix(rawAfterEmoji, "Initial commit")
	isMergeText := strings.HasPrefix(rawAfterEmoji, "Merge ")

	// Initial and merge commits are exempt from the type and emoji requirements.
	if isInitialText {
		return validateInitial(msg, cfg, findings, reporters)
	}
	if isMergeText {
		return validateMerge(msg, cfg, findings, reporters)
	}

	findings = append(findings, validateEmojis(msg, cfg, reporters)...)
	findings = append(findings, validateType(msg, reporters)...)
	findings = append(findings, validateScope(msg, reporters)...)
	findings = append(findings, validateDescription(msg, cfg, reporters)...)
	findings = append(findings, validateBody(msg, cfg, reporters)...)
	findings = append(findings, validateToolTrailers(msg, cfg, reporters)...)

	return findings
}
