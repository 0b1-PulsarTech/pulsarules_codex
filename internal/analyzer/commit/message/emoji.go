package message

import (
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

type emojiWindowConfig struct {
	HardWindow  int
	SoftWindow  int
	Suggestions int
}

func defaultEmojiWindowConfig() emojiWindowConfig {
	const (
		hardWindow  = 5
		softWindow  = 20
		suggestions = 7
	)
	return emojiWindowConfig{
		HardWindow:  hardWindow,
		SoftWindow:  softWindow,
		Suggestions: suggestions,
	}
}

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

func (check emojiCheck) validateWindow() []pass.Finding {
	if len(check.Message.Emojis) == 0 {
		return nil
	}
	current := check.Message.Emojis[0]

	if slices.Contains(check.recentEmojis(check.Config.HardWindow), current) {
		return []pass.Finding{check.Reporters.repeat.NewWithSuggestion(
			"emoji :"+current+": already appears in the last "+
				strconv.Itoa(check.Config.HardWindow)+
				" commits; every commit in that window needs a distinct emoji",
			check.suggestionText(),
		)}
	}
	if slices.Contains(check.recentEmojis(check.Config.SoftWindow), current) {
		return []pass.Finding{check.Reporters.softRepeat.NewWithSuggestion(
			"emoji :"+current+": was used within the last "+strconv.Itoa(check.Config.SoftWindow)+
				" commits; a fresher one would read better",
			check.suggestionText(),
		)}
	}
	return nil
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

func (check emojiCheck) recentEmojis(window int) []string {
	history := check.History
	if window <= 0 || len(history) == 0 {
		return nil
	}
	if len(history) > window {
		// Subjects arrive oldest-first, so the window is the tail of the slice.
		history = history[len(history)-window:]
	}
	shortcodes := make([]string, 0, len(history))
	for _, subject := range history {
		if shortcode := leadingEmoji(subject); shortcode != "" {
			shortcodes = append(shortcodes, shortcode)
		}
	}
	return shortcodes
}

func leadingEmoji(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if !strings.HasPrefix(trimmed, ":") {
		return ""
	}
	name, _, found := strings.Cut(trimmed[1:], ":")
	// An unterminated shortcode swallows the type separator, so a subject like
	// ":wrench feat: X" would otherwise read as the emoji "wrench feat".
	if !found || name == "" || strings.ContainsAny(name, " \t") {
		return ""
	}
	return name
}
