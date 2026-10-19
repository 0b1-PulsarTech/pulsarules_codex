package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type link struct {
	owner string
	ids   []string
}

type reference struct {
	owner  string
	phrase string
	links  func(*knowledge.Index) []link
	exists func(*knowledge.Index) func(id string) bool
}

func ruleLinks(pick func(knowledge.Rule) []string) func(*knowledge.Index) []link {
	return func(idx *knowledge.Index) []link {
		links := make([]link, 0, len(idx.Rules))
		for _, rule := range idx.Rules {
			links = append(links, link{owner: rule.ID, ids: pick(rule)})
		}
		return links
	}
}

func patternLinks(pick func(knowledge.Pattern) []string) func(*knowledge.Index) []link {
	return func(idx *knowledge.Index) []link {
		links := make([]link, 0, len(idx.Patterns))
		for _, pattern := range idx.Patterns {
			links = append(links, link{owner: pattern.ID, ids: pick(pattern)})
		}
		return links
	}
}

func references() []reference {
	return []reference{
		{
			"rule", "depends on unknown rule",
			ruleLinks(func(rule knowledge.Rule) []string { return rule.Dependencies }),
			func(idx *knowledge.Index) func(string) bool { return idx.RuleExists },
		},
		{
			"pattern", "depends on unknown rule",
			patternLinks(func(pattern knowledge.Pattern) []string { return pattern.Dependencies }),
			func(idx *knowledge.Index) func(string) bool { return idx.RuleExists },
		},
		{
			"pattern", "composes unknown pattern",
			patternLinks(func(pattern knowledge.Pattern) []string { return pattern.Composes }),
			func(idx *knowledge.Index) func(string) bool { return idx.PatternExists },
		},
		{
			"rule", "cites unknown reference",
			ruleLinks(func(rule knowledge.Rule) []string { return rule.References }),
			func(idx *knowledge.Index) func(string) bool { return idx.ReferenceExists },
		},
		{
			"pattern", "cites unknown reference",
			patternLinks(func(pattern knowledge.Pattern) []string { return pattern.References }),
			func(idx *knowledge.Index) func(string) bool { return idx.ReferenceExists },
		},
	}
}

func crossReferences(idx *knowledge.Index) []string {
	var problems []string
	for _, kind := range references() {
		exists := kind.exists(idx)
		for _, host := range kind.links(idx) {
			for _, id := range host.ids {
				if !exists(id) {
					problems = append(problems, fmt.Sprintf(
						"%s %q %s %q", kind.owner, host.owner, kind.phrase, id,
					))
				}
			}
		}
	}
	return problems
}
