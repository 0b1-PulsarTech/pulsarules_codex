package commit

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"

// ruleReporters bundles every commit-format sub-reporter declared in
// rules.go, so Validate and its helpers thread one value instead of 14.
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

// defaultRuleReporters returns rules.go's reporters at their compiled-in
// severity, unresolved against any run's config.
func defaultRuleReporters() ruleReporters {
	return ruleReporters{
		initial:         initialReporter,
		merge:           mergeReporter,
		emojiRequired:   emojiRequiredReporter,
		emojiCount:      emojiCountReporter,
		typeRequired:    typeRequiredReporter,
		typeEnum:        typeEnumReporter,
		scopeCharset:    scopeCharsetReporter,
		descRequired:    descRequiredReporter,
		descLength:      descLengthReporter,
		descCapitalize:  descCapitalizeReporter,
		descNoPeriod:    descNoPeriodReporter,
		bodyLength:      bodyLengthReporter,
		bodyTotalLength: bodyTotalLengthReporter,
		noCoauthor:      noCoauthorReporter,
	}
}

// resolved overrides every reporter's severity from ctx's per-analyzer-id
// "severity" param (see core.Reporter.Resolved), so a run's config can
// reach any of the 14 sub-checks without each validate* function spelling
// out the ctx.Params lookup itself.
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

// emojiReporters bundles every emoji.go sub-reporter, resolved the same way.
type emojiReporters struct {
	prohibited core.Reporter
	catalog    core.Reporter
	repeat     core.Reporter
	softRepeat core.Reporter
}

// defaultEmojiReporters returns emoji.go's reporters at their compiled-in
// severity, unresolved against any run's config.
func defaultEmojiReporters() emojiReporters {
	return emojiReporters{
		prohibited: emojiProhibitedReporter,
		catalog:    emojiCatalogReporter,
		repeat:     emojiRepeatReporter,
		softRepeat: emojiSoftRepeatReporter,
	}
}

func (er emojiReporters) resolved(ctx *core.AnalysisContext) emojiReporters {
	er.prohibited = er.prohibited.Resolved(ctx)
	er.catalog = er.catalog.Resolved(ctx)
	er.repeat = er.repeat.Resolved(ctx)
	er.softRepeat = er.softRepeat.Resolved(ctx)
	return er
}
