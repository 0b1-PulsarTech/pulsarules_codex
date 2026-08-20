package commit

import (
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// EmojiWindowConfig holds the repetition windows and the size of the
// suggestion set offered when a rule is broken.
type EmojiWindowConfig struct {
	// HardWindow is how many preceding commits an emoji may not repeat
	// within. A violation blocks the commit.
	HardWindow int
	// SoftWindow is the wider span that only earns advice. A repeat past
	// HardWindow but inside it is reported as information.
	SoftWindow int
	// Suggestions is how many alternatives a finding offers.
	Suggestions int
}

// defaultEmojiWindow is the project default: no repeat within five commits,
// advice up to twenty, seven alternatives offered.
var defaultEmojiWindow = EmojiWindowConfig{HardWindow: 5, SoftWindow: 20, Suggestions: 7}

func DefaultEmojiWindowConfig() EmojiWindowConfig { return defaultEmojiWindow }

// validateWindow enforces emoji variety against recent history. Subjects
// arrive oldest-first, so the window is the TAIL of the slice.
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
