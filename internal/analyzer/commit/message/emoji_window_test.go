package message

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
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
		{"first emoji of a multi-emoji subject", []string{":a::b: feat: X"}, 5, []string{"a"}},
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

			got := emojiCheck{history: testCase.history}.recentEmojis(testCase.window)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("recentEmojis = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestValidateEmoji_SuggestionsExcludeRejected(t *testing.T) {
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
			).validate()
			if len(findings) == 0 {
				t.Fatalf("expected a finding for %q", testCase.subject)
			}
			for _, finding := range findings {
				for _, own := range msg.LeadingEmojis {
					if strings.Contains(finding.Suggestion, ":"+own+":") {
						t.Fatalf("suggestion %q offers back the rejected %q",
							finding.Suggestion, own)
					}
				}
			}
		})
	}
}

func TestHistoryWindow(t *testing.T) {
	t.Parallel()

	defaults := defaultEmojiWindowConfig()
	testCases := []struct {
		name   string
		params pass.ParamSet
		want   int
	}{
		{
			"no params follow the analyzer default",
			nil,
			max(defaults.HardWindow, defaults.SoftWindow),
		},
		{"a hard window above the soft one", pass.ParamSet{
			ParamEmojiHardWindow: 25, ParamEmojiSoftWindow: 10,
		}, 25},
		{"a soft window above the default", pass.ParamSet{ParamEmojiSoftWindow: 30}, 30},
		{"a hard window alone leaves the default soft", pass.ParamSet{
			ParamEmojiHardWindow: 1,
		}, defaults.SoftWindow},
		{"unusable windows fall back to the defaults", pass.ParamSet{
			ParamEmojiHardWindow: 0, ParamEmojiSoftWindow: "x",
		}, defaults.SoftWindow},
		{"negative windows fall back to the defaults", pass.ParamSet{
			ParamEmojiHardWindow: -1, ParamEmojiSoftWindow: -1,
		}, defaults.SoftWindow},
		{"null windows fall back to the defaults", pass.ParamSet{
			ParamEmojiHardWindow: nil, ParamEmojiSoftWindow: nil,
		}, defaults.SoftWindow},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := HistoryWindow(testCase.params); got != testCase.want {
				t.Fatalf("HistoryWindow = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestWindowsAgreeForEveryKeyValue(t *testing.T) {
	t.Parallel()

	defaults := defaultEmojiWindowConfig()
	const explicit = 3
	values := []struct {
		name  string
		value any
		want  func(fallback int) int
	}{
		{"zero", 0, func(fallback int) int { return fallback }},
		{"negative", -2, func(fallback int) int { return fallback }},
		{"null", nil, func(fallback int) int { return fallback }},
		{"string", "x", func(fallback int) int { return fallback }},
		{"float", 3.0, func(int) int { return explicit }},
		{"valid", explicit, func(int) int { return explicit }},
	}
	keys := []struct {
		name     string
		key      string
		fallback int
		other    int
		pick     func(emojiWindowConfig) int
	}{
		{
			"hard", ParamEmojiHardWindow, defaults.HardWindow, defaults.SoftWindow,
			func(cfg emojiWindowConfig) int { return cfg.HardWindow },
		},
		{
			"soft", ParamEmojiSoftWindow, defaults.SoftWindow, defaults.HardWindow,
			func(cfg emojiWindowConfig) int { return cfg.SoftWindow },
		},
	}
	for _, key := range keys {
		for _, value := range values {
			t.Run(key.name+" "+value.name, func(t *testing.T) {
				t.Parallel()

				params := pass.ParamSet{key.key: value.value}
				want := value.want(key.fallback)
				if got := key.pick(defaults.merge(params)); got != want {
					t.Fatalf("check window = %d, want %d", got, want)
				}
				if got := HistoryWindow(params); got != max(want, key.other) {
					t.Fatalf("HistoryWindow = %d, want %d", got, max(want, key.other))
				}
			})
		}
	}
}

func TestDefaultEmojiWindowConfig(t *testing.T) {
	t.Parallel()

	got := defaultEmojiWindowConfig()
	want := emojiWindowConfig{HardWindow: 5, SoftWindow: 20, Suggestions: 7}
	if got != want {
		t.Fatalf("defaultEmojiWindowConfig = %+v, want %+v", got, want)
	}
	if got.SoftWindow <= got.HardWindow {
		t.Fatalf("soft window %d must exceed hard window %d", got.SoftWindow, got.HardWindow)
	}
}
