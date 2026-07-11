package commit

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func TestValidateValidCommit(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: feat(goscan): Detect variable shadowing")
	findings := Validate(msg, DefaultRuleConfig())
	if len(findings) != 0 {
		t.Fatalf("expected no findings for valid commit, got %d: %+v", len(findings), findings)
	}
}

func TestValidateMissingEmoji(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse("feat: Add something")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-emoji-required") {
		t.Errorf("expected commit-emoji-required finding")
	}
}

func TestValidateInvalidType(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: banana: Do something")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-type-enum") {
		t.Errorf("expected commit-type-enum finding")
	}
}

func TestValidateMissingType(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: Do something")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-type-required") {
		t.Errorf("expected commit-type-required finding")
	}
}

func TestValidateScopeCharset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		scope     string
		wantError bool
	}{
		{"valid lowercase", "goscan", false},
		{"valid with underscore", "support_underscore", false},
		{"valid with dash", "auth-ctrl", false},
		{"valid mixed case", "AuthCtrl", false},
		{"invalid with space", "auth ctrl", true},
		{"invalid with ampersand", "support&shield", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			msg := commitmsg.Parse(":wrench: feat(" + testCase.scope + "): Do something")
			findings := Validate(msg, DefaultRuleConfig())
			hasScopeError := hasAnalyzer(findings, "commit-scope-charset")
			if hasScopeError != testCase.wantError {
				t.Errorf("scope charset error = %v, want %v", hasScopeError, testCase.wantError)
			}
		})
	}
}

func TestValidateDescriptionCapitalization(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: feat: add something lowercase")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-desc-capitalize") {
		t.Errorf("expected commit-desc-capitalize finding for lowercase start")
	}
}

func TestValidateDescriptionNoPeriod(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: feat: Add something.")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-desc-no-period") {
		t.Errorf("expected commit-desc-no-period finding")
	}
}

func TestValidateDescriptionTooLong(t *testing.T) {
	t.Parallel()

	var bDesc strings.Builder
	bDesc.WriteString("A")
	for range 80 {
		bDesc.WriteString("x")
	}
	longDesc := bDesc.String()
	msg := commitmsg.Parse(":wrench: feat: " + longDesc)
	findings := Validate(msg, DefaultRuleConfig())
	finding, ok := findingByID(findings, "commit-desc-length")
	if !ok {
		t.Fatalf("expected commit-desc-length finding for 80+ char subject")
	}
	if finding.Severity != core.SeverityError {
		t.Errorf("commit-desc-length severity = %v, want error (must block)", finding.Severity)
	}
}

func TestValidateCoAuthorRejected(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(
		":wrench: feat: Add thing\n\nCo-Authored-By: Claude <noreply@anthropic.com>",
	)
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-no-coauthor") {
		t.Errorf("expected commit-no-coauthor finding")
	}
}

func TestValidateCoAuthorAllowedWhenDisabled(t *testing.T) {
	t.Parallel()

	cfg := DefaultRuleConfig()
	cfg.RejectToolTrailers = false
	msg := commitmsg.Parse(
		":wrench: feat: Add thing\n\nCo-Authored-By: Claude <noreply@anthropic.com>",
	)
	findings := Validate(msg, cfg)
	if hasAnalyzer(findings, "commit-no-coauthor") {
		t.Errorf("should not reject co-author when RejectToolTrailers is false")
	}
}

func TestValidateBodyLineTooLong(t *testing.T) {
	t.Parallel()

	var bLine strings.Builder
	for range 120 {
		bLine.WriteString("x")
	}
	longLine := bLine.String()
	msg := commitmsg.Parse(":wrench: feat: Add thing\n\n" + longLine)
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-body-length") {
		t.Errorf("expected commit-body-length finding")
	}
}

func TestValidateInitialCommit(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":ghost: Initial Commit")
	findings := Validate(msg, DefaultRuleConfig())
	if len(findings) != 0 {
		t.Fatalf("initial commit should be valid, got %d findings: %+v", len(findings), findings)
	}
}

func TestValidateInitialCommitWrongEmoji(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: Initial Commit")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-initial") {
		t.Errorf("expected commit-initial finding for wrong emoji")
	}
}

func TestValidateMergeCommit(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":volcano: Merge branch 'feature-x'")
	findings := Validate(msg, DefaultRuleConfig())
	if len(findings) != 0 {
		t.Fatalf("merge commit should be valid, got %d findings", len(findings))
	}
}

func TestValidateMergeCommitWrongEmoji(t *testing.T) {
	t.Parallel()

	msg := commitmsg.Parse(":wrench: Merge branch 'feature-x'")
	findings := Validate(msg, DefaultRuleConfig())
	if !hasAnalyzer(findings, "commit-merge") {
		t.Errorf("expected commit-merge finding for wrong emoji")
	}
}
