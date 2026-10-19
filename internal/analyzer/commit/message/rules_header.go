package message

import (
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

const (
	initialEmoji = "ghost"
	mergeEmoji   = "volcano"
)

func hasCanonicalEmoji(msg commitmsg.Message) bool {
	switch msg.Kind {
	case commitmsg.Initial:
		return len(msg.LeadingEmojis) == 1 && msg.LeadingEmojis[0] == initialEmoji
	case commitmsg.Merge:
		return len(msg.LeadingEmojis) >= 1 && msg.LeadingEmojis[0] == mergeEmoji
	}
	return false
}

func isScopeChar(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') || char == '-' || char == '_'
}

func validateInitial(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	const requiredInitialSubject = ":" + initialEmoji + ": Initial Commit"
	subject, _, _ := strings.Cut(msg.Raw, "\n")
	if strings.TrimRight(subject, " \t\r") == requiredInitialSubject {
		return nil
	}
	return []finding.Finding{
		reporters[ruleInitial].New(
			"initial commit subject must be exactly '" + requiredInitialSubject + "'",
		),
	}
}

func validateMerge(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	if hasCanonicalEmoji(msg) {
		return nil
	}
	return []finding.Finding{
		reporters[ruleMerge].New("merge commit must start with :" + mergeEmoji + ":"),
	}
}

func validateEmojis(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	var findings []finding.Finding

	if len(msg.LeadingEmojis) == 0 {
		findings = append(
			findings,
			reporters[ruleEmojiRequired].New(
				"commit message must start with an emoji (:shortcode:)",
			),
		)
	}

	if len(msg.LeadingEmojis) > commitmsg.MaxLeadingEmojis {
		findings = append(
			findings,
			reporters[ruleEmojiCount].New(
				"too many leading emojis (max "+strconv.Itoa(commitmsg.MaxLeadingEmojis)+")",
			),
		)
	}

	return findings
}

func validateType(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	if msg.Type == "" {
		return []finding.Finding{
			reporters[ruleTypeRequired].New("commit message must have a Conventional Commit type"),
		}
	}

	if slices.Contains(commitmsg.AllowedTypes(), msg.Type) {
		return nil
	}

	return []finding.Finding{
		reporters[ruleTypeEnum].New(
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
	if strings.ContainsFunc(
		msg.Scope,
		func(char rune) bool { return !isScopeChar(char) },
	) {
		return []finding.Finding{
			reporters[ruleScopeCharset].New(
				"scope must be ASCII alphanumeric with - or _ separators only",
			),
		}
	}
	return nil
}
