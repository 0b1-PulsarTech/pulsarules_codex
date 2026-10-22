package message

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

type emojiCheck struct {
	message   commitmsg.Message
	catalog   *emoji.Catalog
	history   []string
	config    emojiWindowConfig
	reporters ruleReporters
}

func (check emojiCheck) validate() []finding.Finding {
	if hasCanonicalEmoji(check.message) {
		return nil
	}
	return append(check.validateCatalog(), check.validateWindow()...)
}

func (check emojiCheck) validateCatalog() []finding.Finding {
	var findings []finding.Finding
	for _, shortcode := range check.message.LeadingEmojis {
		if replacement, banned := emoji.ProhibitedReplacement(shortcode); banned {
			reason := "emoji :" + shortcode + ": is prohibited by project convention"
			if replacement != "" {
				reason += "; use :" + replacement + ": instead"
			}
			findings = append(
				findings,
				check.reporters[ruleEmojiProhibited].NewWithSuggestion(
					reason,
					check.suggestionText(),
				),
			)
			continue
		}
		// The catalog is the set proven to render in the tools history is read through.
		if !check.catalog.Allows(shortcode) {
			findings = append(findings, check.reporters[ruleEmojiCatalog].NewWithSuggestion(
				"emoji :"+shortcode+": is not in the commit emoji catalog; "+
					"it may not render in the history view",
				check.suggestionText(),
			))
		}
	}
	return findings
}

func (check emojiCheck) suggestionText() string {
	if check.catalog == nil {
		return ""
	}
	// Exclude the window and the emoji already on this message: offering back
	// the one that was just rejected is not an alternative.
	excluded := append(check.recentEmojis(check.config.HardWindow), check.message.LeadingEmojis...)

	seed := check.message.Scope + " " + check.message.Description
	picks := check.catalog.Suggest(
		check.message.Type,
		seed,
		excluded,
		check.config.Suggestions,
	)
	if len(picks) == 0 {
		return ""
	}
	return "try one of: :" + strings.Join(picks, ": :") + ":"
}
