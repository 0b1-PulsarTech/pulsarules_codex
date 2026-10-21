package message

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func TestValidateToolTrailerRejected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		message string
	}{
		{
			"co-authored-by trailer",
			":wrench: feat: Add thing\n\nCo-Authored-By: Claude <noreply@anthropic.com>",
		},
		{
			"claude-session trailer",
			":wrench: feat: Add thing\n\nBody.\n\nClaude-Session: https://example.com/session_x",
		},
		{
			"attribution marker in body",
			":wrench: feat: Add thing\n\nGenerated with Claude Code.",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			findings := validate(
				commitmsg.Parse(testCase.message),
				defaultRuleConfig(),
				defaultRuleReporters(),
			)
			hit, ok := findingByID(findings, "commit-no-coauthor")
			if !ok {
				t.Fatalf("expected commit-no-coauthor finding, got %+v", findings)
			}
			if hit.Severity != finding.SeverityError {
				t.Errorf("severity = %v, want error (must block)", hit.Severity)
			}
		})
	}
}

func TestValidateBodyTotal_TooLongThoughWrapped(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("a line of moderate length that is well under the cap.\n", 10)
	msg := commitmsg.Parse(":wrench: feat: Add thing\n\n" + body)
	findings := validate(msg, defaultRuleConfig(), defaultRuleReporters())
	if hasAnalyzer(findings, "commit-body-length") {
		t.Fatalf("no single body line should exceed the per-line cap: %+v", findings)
	}
	hit, ok := findingByID(findings, "commit-body-total-length")
	if !ok {
		t.Fatalf("expected commit-body-total-length finding for oversized body")
	}
	if hit.Severity != finding.SeverityError {
		t.Errorf("severity = %v, want error (must block)", hit.Severity)
	}
}

func TestValidateShortBodyWithFooterValid(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: feat: Add thing\n\nA short, useful body.\n\nRefs: #42")
	findings := validate(msg, defaultRuleConfig(), defaultRuleReporters())
	if len(findings) != 0 {
		t.Fatalf("short body with a legit footer should be valid, got %+v", findings)
	}
}

func TestValidateSubjectLengthBoundary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		description string
		wantFinding bool
	}{
		{"exactly 72 runes passes", "A" + strings.Repeat("x", 71), false},
		{"73 runes blocks", "A" + strings.Repeat("x", 72), true},
		{"72 multibyte runes pass", "\u00c9" + strings.Repeat("\u00e9", 71), false},
		{"73 multibyte runes block", "\u00c9" + strings.Repeat("\u00e9", 72), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			msg := commitmsg.Parse(":wrench: feat: " + testCase.description)
			findings := validate(msg, defaultRuleConfig(), defaultRuleReporters())
			if got := hasAnalyzer(findings, "commit-desc-length"); got != testCase.wantFinding {
				t.Errorf("commit-desc-length present = %v, want %v", got, testCase.wantFinding)
			}
		})
	}
}
