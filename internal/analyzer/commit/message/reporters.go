package message

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

type ruleReporters struct {
	initial         pass.Reporter
	merge           pass.Reporter
	emojiRequired   pass.Reporter
	emojiCount      pass.Reporter
	typeRequired    pass.Reporter
	typeEnum        pass.Reporter
	scopeCharset    pass.Reporter
	descRequired    pass.Reporter
	descLength      pass.Reporter
	descCapitalize  pass.Reporter
	descNoPeriod    pass.Reporter
	bodyLength      pass.Reporter
	bodyTotalLength pass.Reporter
	noCoauthor      pass.Reporter
}

func commitReporter(id string) pass.Reporter {
	return pass.NewReporter(id, pass.SeverityError, pass.CatCommit)
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

type emojiReporters struct {
	prohibited pass.Reporter
	catalog    pass.Reporter
	repeat     pass.Reporter
	softRepeat pass.Reporter
}

func defaultEmojiReporters() emojiReporters {
	return emojiReporters{
		prohibited: commitReporter("commit-emoji-prohibited"),
		catalog:    commitReporter("commit-emoji-catalog"),
		repeat:     commitReporter("commit-emoji-repeat"),
		softRepeat: pass.NewReporter(
			"commit-emoji-soft-repeat", pass.SeverityInfo, pass.CatCommit,
		),
	}
}
