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

func (rr ruleReporters) resolved(ctx *core.AnalysisContext) ruleReporters {
	rr.initial = rr.initial.Resolved(ctx)
	rr.merge = rr.merge.Resolved(ctx)
	rr.emojiRequired = rr.emojiRequired.Resolved(ctx)
	rr.emojiCount = rr.emojiCount.Resolved(ctx)
	rr.typeRequired = rr.typeRequired.Resolved(ctx)
	rr.typeEnum = rr.typeEnum.Resolved(ctx)
	rr.scopeCharset = rr.scopeCharset.Resolved(ctx)
	rr.descRequired = rr.descRequired.Resolved(ctx)
	rr.descLength = rr.descLength.Resolved(ctx)
	rr.descCapitalize = rr.descCapitalize.Resolved(ctx)
	rr.descNoPeriod = rr.descNoPeriod.Resolved(ctx)
	rr.bodyLength = rr.bodyLength.Resolved(ctx)
	rr.bodyTotalLength = rr.bodyTotalLength.Resolved(ctx)
	rr.noCoauthor = rr.noCoauthor.Resolved(ctx)
	return rr
}

type emojiReporters struct {
	prohibited core.Reporter
	catalog    core.Reporter
	repeat     core.Reporter
	softRepeat core.Reporter
}

func defaultEmojiReporters() emojiReporters {
	return emojiReporters{
		prohibited: commitReporter("commit-emoji-prohibited"),
		catalog:    commitReporter("commit-emoji-catalog"),
		repeat:     commitReporter("commit-emoji-repeat"),
		softRepeat: core.NewReporter(
			"commit-emoji-soft-repeat", core.SeverityInfo, core.CatCommit,
		),
	}
}

func (er emojiReporters) resolved(ctx *core.AnalysisContext) emojiReporters {
	er.prohibited = er.prohibited.Resolved(ctx)
	er.catalog = er.catalog.Resolved(ctx)
	er.repeat = er.repeat.Resolved(ctx)
	er.softRepeat = er.softRepeat.Resolved(ctx)
	return er
}
