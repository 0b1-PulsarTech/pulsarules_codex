package commit

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

// subjects builds an oldest-first history the way gitHistoryEntries yields it.
func subjects(shortcodes ...string) []string {
	history := make([]string, 0, len(shortcodes))
	for _, shortcode := range shortcodes {
		history = append(history, ":"+shortcode+": chore: Something")
	}
	return history
}

func checkFor(t testing.TB, msg commitmsg.Message, history []string) EmojiCheck {
	t.Helper()
	catalog, err := emoji.NewCatalog()
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return EmojiCheck{
		Message: msg,
		Catalog: catalog,
		History: history,
		Config:  DefaultEmojiWindowConfig(),
	}
}

func TestValidateEmojiCatalog(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		subject string
		wantID  string
	}{
		{"catalog member passes", ":wrench: feat: Add routing", ""},
		{"second catalog member passes", ":tea: test: Cover the decoder", ""},
		{"kept exception passes", ":beetle: fix: Squash the parser bug", ""},
		{"off catalog blocks", ":nonexistent_emoji: feat: Add routing", "commit-emoji-catalog"},
		{"country flag blocks", ":us: feat: Add routing", "commit-emoji-catalog"},
		{"non rendering blocks", ":clown_face: feat: Add routing", "commit-emoji-catalog"},
		{"another non rendering blocks", ":abacus: chore: Count things", "commit-emoji-catalog"},
		{"prohibited robot blocks", ":robot: feat: Queue auto sends", "commit-emoji-prohibited"},
		{"prohibited sparkles blocks", ":sparkles: feat: Add routing", "commit-emoji-prohibited"},
		{"prohibited compass blocks", ":compass: feat: Add routing", "commit-emoji-prohibited"},
		{"prohibited test tube blocks", ":test_tube: test: Add cases", "commit-emoji-prohibited"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			findings := checkFor(t, commitmsg.Parse(testCase.subject), nil).ValidateEmoji()
			if testCase.wantID == "" {
				if len(findings) != 0 {
					t.Fatalf("expected no findings, got %+v", findings)
				}
				return
			}

			finding, found := findingByID(findings, testCase.wantID)
			if !found {
				t.Fatalf("expected %s, got %+v", testCase.wantID, findings)
			}
			if finding.Severity != core.SeverityError {
				t.Fatalf("severity = %v, want error", finding.Severity)
			}
			if finding.Suggestion == "" {
				t.Fatal("a blocking emoji finding must offer alternatives")
			}
		})
	}
}

// The replacement is the actionable half of the message; a ban with no way
// forward is what makes a hook infuriating.
func TestValidateEmojiProhibitedNamesItsReplacement(t *testing.T) {
	t.Parallel()

	findings := checkFor(t, commitmsg.Parse(":test_tube: test: Add cases"), nil).ValidateEmoji()
	finding, found := findingByID(findings, "commit-emoji-prohibited")
	if !found {
		t.Fatalf("expected a prohibited finding, got %+v", findings)
	}
	if !strings.Contains(finding.Message, ":tea:") {
		t.Fatalf("message %q does not name the replacement", finding.Message)
	}
}

type windowCase struct {
	name    string
	subject string
	history []string
	wantID  string
}

func windowCases() []windowCase {
	return []windowCase{
		{
			name:    "distinct emoji passes",
			subject: ":wrench: feat: Add routing",
			history: subjects("memo", "tea", "bug", "gear", "package"),
			wantID:  "",
		},
		{
			name:    "immediately preceding repeat blocks",
			subject: ":wrench: feat: Add routing",
			history: subjects("memo", "tea", "bug", "gear", "wrench"),
			wantID:  "commit-emoji-repeat",
		},
		{
			name:    "repeat at the far edge of the window blocks",
			subject: ":wrench: feat: Add routing",
			history: subjects("memo", "wrench", "bug", "gear", "package"),
			wantID:  "commit-emoji-repeat",
		},
		{
			name:    "repeat just outside the hard window only advises",
			subject: ":wrench: feat: Add routing",
			history: subjects("wrench", "memo", "tea", "bug", "gear", "package"),
			wantID:  "commit-emoji-soft-repeat",
		},
		{
			name:    "repeat outside every window passes",
			subject: ":wrench: feat: Add routing",
			history: subjects("wrench", "memo", "tea", "bug", "gear", "package",
				"hammer", "whale", "art", "skull", "seedling", "boar",
				"ram", "factory", "sailboat", "pager", "label", "lock",
				"key", "cloud", "satellite"),
			wantID: "",
		},
		{
			name:    "empty history passes",
			subject: ":wrench: feat: Add routing",
			history: nil,
			wantID:  "",
		},
		{
			name:    "initial commit is exempt",
			subject: ":ghost: Initial Commit",
			history: subjects("ghost"),
			wantID:  "",
		},
		{
			name:    "merge commit is exempt",
			subject: ":volcano: Merge branch 'x'",
			history: subjects("volcano"),
			wantID:  "",
		},
	}
}

func TestValidateEmojiWindow(t *testing.T) {
	t.Parallel()

	for _, testCase := range windowCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			findings := checkFor(
				t,
				commitmsg.Parse(testCase.subject),
				testCase.history,
			).ValidateEmoji()
			if testCase.wantID == "" {
				if len(findings) != 0 {
					t.Fatalf("expected no findings, got %+v", findings)
				}
				return
			}

			finding, found := findingByID(findings, testCase.wantID)
			if !found {
				t.Fatalf("expected %s, got %+v", testCase.wantID, findings)
			}
			wantSeverity := core.SeverityError
			if testCase.wantID == "commit-emoji-soft-repeat" {
				wantSeverity = core.SeverityInfo
			}
			if finding.Severity != wantSeverity {
				t.Fatalf("severity = %v, want %v", finding.Severity, wantSeverity)
			}
		})
	}
}

// History arrives oldest-first. Reading the window off the head instead of the
// tail silently checks the oldest commits and lets fresh repeats through.
func TestValidateEmojiWindowReadsTheNewestEnd(t *testing.T) {
	t.Parallel()

	// The repeat sits in the newest five; the six oldest entries are filler.
	history := subjects("memo", "tea", "bug", "gear", "package", "art",
		"wrench", "hammer", "whale", "skull", "seedling")

	findings := checkFor(t, commitmsg.Parse(":wrench: feat: Add routing"), history).ValidateEmoji()
	if !hasAnalyzer(findings, "commit-emoji-repeat") {
		t.Fatalf("expected a hard repeat from the newest end, got %+v", findings)
	}
}

// Acting on the advice must not break the very rule that produced it.
func TestValidateEmojiSuggestionsAvoidTheWindow(t *testing.T) {
	t.Parallel()

	history := subjects("wrench", "tea", "bug", "gear", "package")
	findings := checkFor(t, commitmsg.Parse(":wrench: feat: Add routing"), history).ValidateEmoji()

	finding, found := findingByID(findings, "commit-emoji-repeat")
	if !found {
		t.Fatalf("expected a repeat finding, got %+v", findings)
	}
	for _, windowed := range []string{"wrench", "tea", "bug", "gear", "package"} {
		if strings.Contains(finding.Suggestion, ":"+windowed+":") {
			t.Fatalf("suggestion %q offers windowed emoji %q", finding.Suggestion, windowed)
		}
	}
}

func TestValidateEmojiChecksEveryLeadingEmoji(t *testing.T) {
	t.Parallel()

	findings := checkFor(
		t,
		commitmsg.Parse(":wrench: :robot: feat: Add routing"),
		nil,
	).ValidateEmoji()
	if !hasAnalyzer(findings, "commit-emoji-prohibited") {
		t.Fatalf("expected the trailing prohibited emoji to be caught, got %+v", findings)
	}
}
