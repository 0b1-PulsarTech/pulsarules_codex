package commit

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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
			findings := Validate(commitmsg.Parse(testCase.message), DefaultRuleConfig())
			finding, ok := findingByID(findings, "commit-no-coauthor")
			if !ok {
				t.Fatalf("expected commit-no-coauthor finding, got %+v", findings)
			}
			if finding.Severity != core.SeverityError {
				t.Errorf("severity = %v, want error (must block)", finding.Severity)
			}
		})
	}
}

func TestValidateBodyTotalTooLong(t *testing.T) {
	t.Parallel()

	// Every line stays under MaxBodyLineLen, but the total exceeds MaxBodyLen,
	// reproducing the oversized-but-wrapped body that previously slipped through.
	body := strings.Repeat("a line of moderate length that is well under the cap.\n", 10)
	msg := commitmsg.Parse(":wrench: feat: Add thing\n\n" + body)
	findings := Validate(msg, DefaultRuleConfig())
	if hasAnalyzer(findings, "commit-body-length") {
		t.Fatalf("no single body line should exceed the per-line cap: %+v", findings)
	}
	finding, ok := findingByID(findings, "commit-body-total-length")
	if !ok {
		t.Fatalf("expected commit-body-total-length finding for oversized body")
	}
	if finding.Severity != core.SeverityError {
		t.Errorf("severity = %v, want error (must block)", finding.Severity)
	}
}

func TestValidateShortBodyWithFooterValid(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: feat: Add thing\n\nA short, useful body.\n\nRefs: #42")
	findings := Validate(msg, DefaultRuleConfig())
	if len(findings) != 0 {
		t.Fatalf("short body with a legit footer should be valid, got %+v", findings)
	}
}
