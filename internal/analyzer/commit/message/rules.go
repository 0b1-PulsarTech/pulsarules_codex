package message

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

type ruleConfig struct {
	MaxSubjectLen  int
	MaxBodyLineLen int
	MaxBodyLen     int
}

const typicalFindingCapacity = 8

func defaultRuleConfig() ruleConfig {
	const (
		defaultMaxSubjectLen  = 72
		defaultMaxBodyLineLen = 100
		defaultMaxBodyLen     = 150
	)
	return ruleConfig{
		MaxSubjectLen:  defaultMaxSubjectLen,
		MaxBodyLineLen: defaultMaxBodyLineLen,
		MaxBodyLen:     defaultMaxBodyLen,
	}
}

func validate(msg commitmsg.Message, cfg ruleConfig, reporters ruleReporters) []finding.Finding {
	findings := make([]finding.Finding, 0, typicalFindingCapacity)

	// Initial and merge commits are exempt from the type and emoji requirements.
	switch msg.Kind {
	case commitmsg.Initial:
		return validateInitial(msg, reporters)
	case commitmsg.Merge:
		return validateMerge(msg, reporters)
	case commitmsg.Regular:
	}

	findings = append(findings, validateEmojis(msg, reporters)...)
	findings = append(findings, validateType(msg, reporters)...)
	findings = append(findings, validateScope(msg, reporters)...)
	findings = append(findings, validateDescription(msg, cfg, reporters)...)
	findings = append(findings, validateBody(msg, cfg, reporters)...)
	findings = append(findings, validateToolTrailers(msg, reporters)...)

	return findings
}
