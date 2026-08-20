package commit

import (
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type EmojiWindowConfig struct {
	HardWindow  int
	SoftWindow  int
	Suggestions int
}

func DefaultEmojiWindowConfig() EmojiWindowConfig {
	const (
		hardWindow  = 5
		softWindow  = 20
		suggestions = 7
	)
	return EmojiWindowConfig{
		HardWindow:  hardWindow,
		SoftWindow:  softWindow,
		Suggestions: suggestions,
	}
}

func (check EmojiCheck) validateWindow() []core.Finding {
	if len(check.Message.Emojis) == 0 {
		return nil
	}
	current := check.Message.Emojis[0]

	if slices.Contains(check.recentEmojis(check.Config.HardWindow), current) {
		return []core.Finding{check.Reporters.repeat.NewWithSuggestion(
			"emoji :"+current+": already appears in the last "+
				strconv.Itoa(check.Config.HardWindow)+
				" commits; every commit in that window needs a distinct emoji",
			check.suggestionText(),
		)}
	}
	if slices.Contains(check.recentEmojis(check.Config.SoftWindow), current) {
		return []core.Finding{check.Reporters.softRepeat.NewWithSuggestion(
			"emoji :"+current+": was used within the last "+strconv.Itoa(check.Config.SoftWindow)+
				" commits; a fresher one would read better",
			check.suggestionText(),
		)}
	}
	return nil
}

func (check EmojiCheck) recentEmojis(window int) []string {
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
