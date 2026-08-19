package commit

import (
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
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

var (
	emojiProhibitedReporter = core.NewReporter(
		"commit-emoji-prohibited",
		core.SeverityError,
		core.CatCommit,
	)
	emojiCatalogReporter = core.NewReporter(
		"commit-emoji-catalog",
		core.SeverityError,
		core.CatCommit,
	)
	emojiRepeatReporter = core.NewReporter(
		"commit-emoji-repeat",
		core.SeverityError,
		core.CatCommit,
	)
	emojiSoftRepeatReporter = core.NewReporter(
		"commit-emoji-soft-repeat",
		core.SeverityInfo,
		core.CatCommit,
	)
)

// EmojiCheck is one commit message weighed against the catalog and the recent
// history. History holds recent commit subjects OLDEST FIRST, as git log
// yields them once reversed. Reporters carries the 4 sub-rule reporters,
// already resolved against the run's config (see emojiReporters.resolved).
type EmojiCheck struct {
	Message   commitmsg.Message
	Catalog   *emoji.Catalog
	History   []string
	Config    EmojiWindowConfig
	Reporters emojiReporters
}

// ValidateEmoji reports every emoji rule the commit breaks: membership of the
// catalog, the ban list, and repetition against recent history.
func (check EmojiCheck) ValidateEmoji() []core.Finding {
	if check.Message.IsInitial || check.Message.IsMerge {
		return nil
	}
	return append(check.validateCatalog(), check.validateWindow()...)
}

// validateCatalog rejects emoji outside the catalog because the catalog is the
// set proven to render in the tools this history is read through, and rejects
// banned ones by project decision.
func (check EmojiCheck) validateCatalog() []core.Finding {
	var findings []core.Finding
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
		if !check.Catalog.Allows(shortcode) {
			findings = append(findings, check.Reporters.catalog.NewWithSuggestion(
				"emoji :"+shortcode+": is not in the commit emoji catalog; it may not render in the history view",
				check.suggestionText(),
			))
		}
	}
	return findings
}

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

// suggestionText offers alternatives that are in the catalog and absent from
// the hard window, so acting on the advice cannot break another rule.
func (check EmojiCheck) suggestionText() string {
	if check.Catalog == nil {
		return ""
	}
	// Exclude the window AND the emoji already on this message: offering back
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
