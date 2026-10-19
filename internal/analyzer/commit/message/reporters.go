package message

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

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

func (reporters ruleReporters) resolved(ctx *pass.Context) ruleReporters {
	for _, reporter := range reporters.all() {
		*reporter = pass.ResolveReporter(*reporter, ctx)
	}
	return reporters
}

func (reporters *ruleReporters) all() []*finding.Reporter {
	return []*finding.Reporter{
		&reporters.initial, &reporters.merge, &reporters.emojiRequired, &reporters.emojiCount,
		&reporters.typeRequired, &reporters.typeEnum, &reporters.scopeCharset,
		&reporters.descRequired, &reporters.descLength, &reporters.descCapitalize,
		&reporters.descNoPeriod, &reporters.bodyLength, &reporters.bodyTotalLength,
		&reporters.noCoauthor,
	}
}

type emojiReporters struct {
	prohibited finding.Reporter
	catalog    finding.Reporter
	repeat     finding.Reporter
	softRepeat finding.Reporter
}

func defaultEmojiReporters() emojiReporters {
	return emojiReporters{
		prohibited: commitReporter("commit-emoji-prohibited"),
		catalog:    commitReporter("commit-emoji-catalog"),
		repeat:     commitReporter("commit-emoji-repeat"),
		softRepeat: finding.NewReporter(
			"commit-emoji-soft-repeat", finding.SeverityInfo, finding.CatCommit,
		),
	}
}

func (reporters emojiReporters) resolved(ctx *pass.Context) emojiReporters {
	for _, reporter := range []*finding.Reporter{
		&reporters.prohibited, &reporters.catalog, &reporters.repeat, &reporters.softRepeat,
	} {
		*reporter = pass.ResolveReporter(*reporter, ctx)
	}
	return reporters
}
