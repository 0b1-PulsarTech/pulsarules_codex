package commit

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

// RuleConfig holds the configurable thresholds for commit validation.
type RuleConfig struct {
	// MaxSubjectLen is the maximum subject line length. Default: 72.
	MaxSubjectLen int
	// MaxBodyLineLen is the maximum body line length. Default: 100.
	MaxBodyLineLen int
	// MaxBodyLen is the maximum total body length in runes. Default: 150.
	MaxBodyLen int
	// RequireEmoji controls whether a leading emoji is required.
	RequireEmoji bool
	// RejectToolTrailers controls whether tool-attribution trailers
	// (Co-Authored-By, Claude-Session, ...) are rejected.
	RejectToolTrailers bool
}

const (
	defaultMaxSubjectLen  = 72
	defaultMaxBodyLineLen = 100
	defaultMaxBodyLen     = 150

	// typicalFindingCapacity sizes a findings slice for the common case: a
	// commit trips at most a handful of the rule/emoji checks, so this
	// avoids slice growth without over-allocating for the rare bad commit.
	typicalFindingCapacity = 8
)

// DefaultRuleConfig returns the default commit validation thresholds.
func DefaultRuleConfig() RuleConfig {
	return RuleConfig{
		MaxSubjectLen:      defaultMaxSubjectLen,
		MaxBodyLineLen:     defaultMaxBodyLineLen,
		MaxBodyLen:         defaultMaxBodyLen,
		RequireEmoji:       true,
		RejectToolTrailers: true,
	}
}

var (
	initialReporter       = core.NewReporter("commit-initial", core.SeverityError, core.CatCommit)
	mergeReporter         = core.NewReporter("commit-merge", core.SeverityError, core.CatCommit)
	emojiRequiredReporter = core.NewReporter(
		"commit-emoji-required",
		core.SeverityError,
		core.CatCommit,
	)
	emojiCountReporter = core.NewReporter(
		"commit-emoji-count",
		core.SeverityError,
		core.CatCommit,
	)
	typeRequiredReporter = core.NewReporter(
		"commit-type-required",
		core.SeverityError,
		core.CatCommit,
	)
	typeEnumReporter = core.NewReporter(
		"commit-type-enum",
		core.SeverityError,
		core.CatCommit,
	)
	scopeCharsetReporter = core.NewReporter(
		"commit-scope-charset",
		core.SeverityError,
		core.CatCommit,
	)
	descRequiredReporter = core.NewReporter(
		"commit-desc-required",
		core.SeverityError,
		core.CatCommit,
	)
	descLengthReporter = core.NewReporter(
		"commit-desc-length",
		core.SeverityError,
		core.CatCommit,
	)
	descCapitalizeReporter = core.NewReporter(
		"commit-desc-capitalize",
		core.SeverityError,
		core.CatCommit,
	)
	descNoPeriodReporter = core.NewReporter(
		"commit-desc-no-period",
		core.SeverityError,
		core.CatCommit,
	)
	bodyLengthReporter = core.NewReporter(
		"commit-body-length",
		core.SeverityError,
		core.CatCommit,
	)
	bodyTotalLengthReporter = core.NewReporter(
		"commit-body-total-length",
		core.SeverityError,
		core.CatCommit,
	)
	noCoauthorReporter = core.NewReporter(
		"commit-no-coauthor",
		core.SeverityError,
		core.CatCommit,
	)
)

// Validate checks a parsed commit message against all rules and returns
// findings. Special cases (initial commit, merge, WIP) are exempt from
// type/emoji requirements. reporters carries the 14 sub-rule reporters,
// already resolved against the run's config (see ruleReporters.resolved).
func Validate(msg commitmsg.Message, cfg RuleConfig, reporters ruleReporters) []core.Finding {
	findings := make([]core.Finding, 0, typicalFindingCapacity)

	rawAfterEmoji := strings.TrimSpace(msg.Raw)
	for _, e := range msg.Emojis {
		rawAfterEmoji = strings.TrimSpace(strings.TrimPrefix(rawAfterEmoji, ":"+e+":"))
	}

	isInitialText := strings.HasPrefix(rawAfterEmoji, "Initial Commit") ||
		strings.HasPrefix(rawAfterEmoji, "Initial commit")
	isMergeText := strings.HasPrefix(rawAfterEmoji, "Merge ")

	if isInitialText {
		return validateInitial(msg, cfg, findings, reporters)
	}
	if isMergeText {
		return validateMerge(msg, cfg, findings, reporters)
	}

	findings = append(findings, validateEmojis(msg, cfg, reporters)...)
	findings = append(findings, validateType(msg, reporters)...)
	findings = append(findings, validateScope(msg, reporters)...)
	findings = append(findings, validateDescription(msg, cfg, reporters)...)
	findings = append(findings, validateBody(msg, cfg, reporters)...)
	findings = append(findings, validateToolTrailers(msg, cfg, reporters)...)

	return findings
}
