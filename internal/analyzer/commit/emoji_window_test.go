package commit

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func TestRecentEmojis(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		history []string
		window  int
		want    []string
	}{
		{"tail of a longer history", subjects("a", "b", "c"), 2, []string{"b", "c"}},
		{"whole history when shorter", subjects("a", "b"), 5, []string{"a", "b"}},
		{"zero window", subjects("a", "b"), 0, nil},
		{"negative window", subjects("a", "b"), -1, nil},
		{"empty history", nil, 5, nil},
		{
			"subjects without emoji are skipped",
			[]string{"no emoji here", ":a: chore: X"},
			5,
			[]string{"a"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := EmojiCheck{History: testCase.history}.recentEmojis(testCase.window)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("recentEmojis = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestLeadingEmoji(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		subject string
		want    string
	}{
		{"plain shortcode", ":wrench: feat: Add", "wrench"},
		{"leading whitespace", "  :tea: test: Add", "tea"},
		{"no emoji", "feat: Add", ""},
		{"unterminated shortcode", ":wrench feat: Add", ""},
		{"empty subject", "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := leadingEmoji(testCase.subject); got != testCase.want {
				t.Fatalf("leadingEmoji(%q) = %q, want %q", testCase.subject, got, testCase.want)
			}
		})
	}
}

// Offering back the emoji that was just rejected is not an alternative.
func TestValidateEmojiSuggestionsExcludeTheRejectedEmoji(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		subject string
	}{
		{"prohibited", ":sparkles: feat: Add the lead importer"},
		{"off catalog", ":nonexistent_emoji: test: Cover the decoder"},
		{"repeat", ":tea: test: Cover the decoder"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			msg := commitmsg.Parse(testCase.subject)
			findings := checkFor(
				t,
				msg,
				subjects("tea", "memo", "bug", "gear", "package"),
			).ValidateEmoji()
			if len(findings) == 0 {
				t.Fatalf("expected a finding for %q", testCase.subject)
			}
			for _, finding := range findings {
				for _, own := range msg.Emojis {
					if strings.Contains(finding.Suggestion, ":"+own+":") {
						t.Fatalf("suggestion %q offers back the rejected %q",
							finding.Suggestion, own)
					}
				}
			}
		})
	}
}
