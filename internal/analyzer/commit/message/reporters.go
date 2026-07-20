package message

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

type ruleReporters struct {
	initial         finding.Reporter
	merge           finding.Reporter
	emojiRequired   finding.Reporter
	emojiCount      finding.Reporter
	typeRequired    finding.Reporter
	typeEnum        finding.Reporter
	scopeCharset    finding.Reporter
	descRequired    finding.Reporter
	descLength      finding.Reporter
	descCapitalize  finding.Reporter
	descNoPeriod    finding.Reporter
	bodyLength      finding.Reporter
	bodyTotalLength finding.Reporter
	noCoauthor      finding.Reporter
}

func commitReporter(id string) finding.Reporter {
	return finding.NewReporter(id, finding.SeverityError, finding.CatCommit)
}

func defaultRuleReporters() ruleReporters {
	return ruleReporters{
		initial:         commitReporter("commit-initial"),
		merge:           commitReporter("commit-merge"),
		emojiRequired:   commitReporter("commit-emoji-required"),
		emojiCount:      commitReporter("commit-emoji-count"),
		typeRequired:    commitReporter("commit-type-required"),
		typeEnum:        commitReporter("commit-type-enum"),
		scopeCharset:    commitReporter("commit-scope-charset"),
		descRequired:    commitReporter("commit-desc-required"),
		descLength:      commitReporter("commit-desc-length"),
		descCapitalize:  commitReporter("commit-desc-capitalize"),
		descNoPeriod:    commitReporter("commit-desc-no-period"),
		bodyLength:      commitReporter("commit-body-length"),
		bodyTotalLength: commitReporter("commit-body-total-length"),
		noCoauthor:      commitReporter("commit-no-coauthor"),
	}
}
