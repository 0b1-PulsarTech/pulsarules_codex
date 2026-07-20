package message

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

type ruleConfig struct {
	MaxSubjectLen      int
	MaxBodyLineLen     int
	MaxBodyLen         int
	RequireEmoji       bool
	RejectToolTrailers bool
}

func DefaultRuleConfig() ruleConfig {
	return ruleConfig{
		MaxSubjectLen:      72,
		MaxBodyLineLen:     100,
		MaxBodyLen:         150,
		RequireEmoji:       true,
		RejectToolTrailers: true,
	}
}

func Validate(msg commitmsg.Message, cfg ruleConfig, reporters ruleReporters) []finding.Finding {
	findings := make([]finding.Finding, 0, 8)

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
