package message

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type rule int

const (
	ruleInitial rule = iota
	ruleMerge
	ruleEmojiRequired
	ruleEmojiCount
	ruleTypeRequired
	ruleTypeEnum
	ruleScopeCharset
	ruleDescRequired
	ruleDescLength
	ruleDescCapitalize
	ruleDescNoPeriod
	ruleBodyLength
	ruleBodyTotalLength
	ruleNoCoauthor
	ruleEmojiProhibited
	ruleEmojiCatalog
	ruleEmojiRepeat
	ruleEmojiSoftRepeat
	ruleCount
)

type ruleSpec struct {
	id       string
	severity finding.Severity
}

type ruleReporters [ruleCount]finding.Reporter

func ruleSpecs() [ruleCount]ruleSpec {
	return [ruleCount]ruleSpec{
		ruleInitial:         {"commit-initial", finding.SeverityError},
		ruleMerge:           {"commit-merge", finding.SeverityError},
		ruleEmojiRequired:   {"commit-emoji-required", finding.SeverityError},
		ruleEmojiCount:      {"commit-emoji-count", finding.SeverityError},
		ruleTypeRequired:    {"commit-type-required", finding.SeverityError},
		ruleTypeEnum:        {"commit-type-enum", finding.SeverityError},
		ruleScopeCharset:    {"commit-scope-charset", finding.SeverityError},
		ruleDescRequired:    {"commit-desc-required", finding.SeverityError},
		ruleDescLength:      {"commit-desc-length", finding.SeverityError},
		ruleDescCapitalize:  {"commit-desc-capitalize", finding.SeverityError},
		ruleDescNoPeriod:    {"commit-desc-no-period", finding.SeverityError},
		ruleBodyLength:      {"commit-body-length", finding.SeverityError},
		ruleBodyTotalLength: {"commit-body-total-length", finding.SeverityError},
		ruleNoCoauthor:      {"commit-no-coauthor", finding.SeverityError},
		ruleEmojiProhibited: {"commit-emoji-prohibited", finding.SeverityError},
		ruleEmojiCatalog:    {"commit-emoji-catalog", finding.SeverityError},
		ruleEmojiRepeat:     {"commit-emoji-repeat", finding.SeverityError},
		ruleEmojiSoftRepeat: {"commit-emoji-soft-repeat", finding.SeverityInfo},
	}
}

func defaultRuleReporters() ruleReporters {
	var reporters ruleReporters
	for index, spec := range ruleSpecs() {
		reporters[index] = finding.NewReporter(spec.id, spec.severity, finding.CatCommit)
	}
	return reporters
}

func (reporters ruleReporters) resolved(ctx *pass.Context) ruleReporters {
	for index := range reporters {
		reporters[index] = pass.ResolveReporter(reporters[index], ctx)
	}
	return reporters
}
