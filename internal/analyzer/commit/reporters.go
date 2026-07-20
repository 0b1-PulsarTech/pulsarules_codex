package commit

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"

type ruleReporters struct {
	initial         core.Reporter
	merge           core.Reporter
	emojiRequired   core.Reporter
	emojiCount      core.Reporter
	typeRequired    core.Reporter
	typeEnum        core.Reporter
	scopeCharset    core.Reporter
	descRequired    core.Reporter
	descLength      core.Reporter
	descCapitalize  core.Reporter
	descNoPeriod    core.Reporter
	bodyLength      core.Reporter
	bodyTotalLength core.Reporter
	noCoauthor      core.Reporter
}

func commitReporter(id string) core.Reporter {
	return core.NewReporter(id, core.SeverityError, core.CatCommit)
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
