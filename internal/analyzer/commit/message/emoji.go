package message

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

type emojiCheck struct {
	Message   commitmsg.Message
	Catalog   *emoji.Catalog
	History   []string
	Config    emojiWindowConfig
	Reporters emojiReporters
}

func (check emojiCheck) ValidateEmoji() []pass.Finding {
	if check.Message.IsInitial || check.Message.IsMerge {
		return nil
	}
	return append(check.validateCatalog(), check.validateWindow()...)
}

func (check emojiCheck) validateCatalog() []pass.Finding {
	var findings []pass.Finding
	for _, shortcode := range check.Message.Emojis {
		if replacement, banned := emoji.ProhibitedReplacement(shortcode); banned {
			message := "emoji :" + shortcode + ": is prohibited by project convention"
			if replacement != "" {
				message += "; use :" + replacement + ": instead"
			}
			findings = append(
				findings,
				check.Reporters.prohibited.NewWithSuggestion(message, check.suggestionText()),
			)
			continue
		}
		// The catalog is the set proven to render in the tools history is read through.
		if !check.Catalog.Allows(shortcode) {
			findings = append(findings, check.Reporters.catalog.NewWithSuggestion(
				"emoji :"+shortcode+": is not in the commit emoji catalog; "+
					"it may not render in the history view",
				check.suggestionText(),
			))
		}
	}
	return findings
}

func (check emojiCheck) suggestionText() string {
	if check.Catalog == nil {
		return ""
	}
	// Exclude the window and the emoji already on this message: offering back
	// the one that was just rejected is not an alternative.
	excluded := append(check.recentEmojis(check.Config.HardWindow), check.Message.Emojis...)

	seed := check.Message.Scope + " " + check.Message.Description
	picks := check.Catalog.Suggest(
		check.Message.Type,
		seed,
		excluded,
		check.Config.Suggestions,
	)
	if len(picks) == 0 {
		return ""
	}
	return "try one of: :" + strings.Join(picks, ": :") + ":"
}
