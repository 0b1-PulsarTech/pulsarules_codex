package message

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func validateInitial(
	msg commitmsg.Message, _ ruleConfig, findings []finding.Finding, reporters ruleReporters,
) []finding.Finding {
	if len(msg.Emojis) != 1 || msg.Emojis[0] != "ghost" {
		findings = append(
			findings,
			reporters.initial.New("initial commit must be :ghost: Initial Commit"),
		)
	}
	return findings
}

func validateMerge(
	msg commitmsg.Message, _ ruleConfig, findings []finding.Finding, reporters ruleReporters,
) []finding.Finding {
	if len(msg.Emojis) < 1 || msg.Emojis[0] != "volcano" {
		findings = append(findings, reporters.merge.New("merge commit must start with :volcano:"))
	}
	return findings
}

func validateEmojis(
	msg commitmsg.Message, cfg ruleConfig, reporters ruleReporters,
) []finding.Finding {
	var findings []finding.Finding

	if cfg.RequireEmoji && len(msg.Emojis) == 0 {
		findings = append(
			findings,
			reporters.emojiRequired.New("commit message must start with an emoji (:shortcode:)"),
		)
	}

	if len(msg.Emojis) > 3 {
		findings = append(findings, reporters.emojiCount.New("too many leading emojis (max 3)"))
	}

	return findings
}

func validateType(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	if msg.Type == "" {
		return []finding.Finding{
			reporters.typeRequired.New("commit message must have a Conventional Commit type"),
		}
	}

	if slices.Contains(commitmsg.AllowedTypes(), msg.Type) {
		return nil
	}

	return []finding.Finding{
		reporters.typeEnum.New(
			"type '" + msg.Type + "' is not allowed; use one of: " + strings.Join(
				commitmsg.AllowedTypes(),
				", ",
			),
		),
	}
}

func validateScope(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	if msg.Scope == "" {
		return nil
	}
	for _, char := range msg.Scope {
		if !isScopeChar(char) {
			return []finding.Finding{
				reporters.scopeCharset.New(
					"scope must be ASCII alphanumeric with - or _ separators only",
				),
			}
		}
	}
	return nil
}
