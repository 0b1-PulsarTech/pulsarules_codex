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

func (reporters ruleReporters) resolved(ctx *core.AnalysisContext) ruleReporters {
	reporters.initial = reporters.initial.Resolved(ctx)
	reporters.merge = reporters.merge.Resolved(ctx)
	reporters.emojiRequired = reporters.emojiRequired.Resolved(ctx)
	reporters.emojiCount = reporters.emojiCount.Resolved(ctx)
	reporters.typeRequired = reporters.typeRequired.Resolved(ctx)
	reporters.typeEnum = reporters.typeEnum.Resolved(ctx)
	reporters.scopeCharset = reporters.scopeCharset.Resolved(ctx)
	reporters.descRequired = reporters.descRequired.Resolved(ctx)
	reporters.descLength = reporters.descLength.Resolved(ctx)
	reporters.descCapitalize = reporters.descCapitalize.Resolved(ctx)
	reporters.descNoPeriod = reporters.descNoPeriod.Resolved(ctx)
	reporters.bodyLength = reporters.bodyLength.Resolved(ctx)
	reporters.bodyTotalLength = reporters.bodyTotalLength.Resolved(ctx)
	reporters.noCoauthor = reporters.noCoauthor.Resolved(ctx)
	return reporters
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

func (reporters emojiReporters) resolved(ctx *core.AnalysisContext) emojiReporters {
	reporters.prohibited = reporters.prohibited.Resolved(ctx)
	reporters.catalog = reporters.catalog.Resolved(ctx)
	reporters.repeat = reporters.repeat.Resolved(ctx)
	reporters.softRepeat = reporters.softRepeat.Resolved(ctx)
	return reporters
}
