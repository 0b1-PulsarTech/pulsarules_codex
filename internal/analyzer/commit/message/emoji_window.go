package message

import (
	"slices"
	"strconv"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
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

func (check emojiCheck) validateWindow() []finding.Finding {
	if len(check.message.LeadingEmojis) == 0 {
		return nil
	}
	current := check.message.LeadingEmojis[0]

	if slices.Contains(check.recentEmojis(check.config.HardWindow), current) {
		return []finding.Finding{check.reporters[ruleEmojiRepeat].NewWithSuggestion(
			"emoji :"+current+": already appears in the last "+
				strconv.Itoa(check.config.HardWindow)+
				" commits; every commit in that window needs a distinct emoji",
			check.suggestionText(),
		)}
	}
	if slices.Contains(check.recentEmojis(check.config.SoftWindow), current) {
		return []finding.Finding{check.reporters[ruleEmojiSoftRepeat].NewWithSuggestion(
			"emoji :"+current+": was used within the last "+strconv.Itoa(check.config.SoftWindow)+
				" commits; a fresher one would read better",
			check.suggestionText(),
		)}
	}
	return nil
}

func (check emojiCheck) recentEmojis(window int) []string {
	history := check.history
	if window <= 0 || len(history) == 0 {
		return nil
	}
	if len(history) > window {
		// Subjects arrive oldest-first, so the window is the tail of the slice.
		history = history[len(history)-window:]
	}
	shortcodes := make([]string, 0, len(history))
	for _, subject := range history {
		if emojis := commitmsg.Parse(subject).LeadingEmojis; len(emojis) > 0 {
			shortcodes = append(shortcodes, emojis[0])
		}
	}
	return shortcodes
}
