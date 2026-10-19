package render

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type sectionSpec struct{ Key, Heading string }

func canonicalSections() []sectionSpec {
	return []sectionSpec{
		{"when", "When this applies"},
		{"catalog", "The catalog"},
		{"must", "Must"},
		{"recipe", "Recipe"},
		{"approved", "Approved patterns and where they appear"},
		{"rejected", "Rejected anti-patterns (and the rule each breaks)"},
		{"examples", "Examples"},
		{"forbidden", "Forbidden"},
		{"validation", "Validation checklist"},
		// Outputs render last: they summarize the payoff of everything above.
		{"outputs", "Expected outputs"},
	}
}

type contribution struct {
	Name string
	Body string
}

type mergedSection struct {
	Heading string
	Items   []contribution
}

func mergeSources(
	parser *bodyParser, idx *knowledge.Index, skill knowledge.Skill,
) ([]source, error) {
	var sources []source
	for entry := range composed(idx, skill.ComposeRules, skill.ComposePatterns) {
		if !entry.Found {
			return nil, fmt.Errorf(
				"%w: skill %q composes unknown %s %q",
				ErrUnknownComposition, skill.ID, entry.Label, entry.ID,
			)
		}
		sections, err := parser.sections(idx.Body(entry.Kind, entry.ID))
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", entry.Kind, entry.ID, err)
		}
		sources = append(sources, source{name: entry.Name, sections: sections})
	}
	return sources, nil
}

func normativeSectionKeys() []string {
	return []string{"must", "forbidden", "validation"}
}

type source struct {
	name     string
	sections map[string]string
}

func mergeSections(sources []source) []mergedSection {
	var merged []mergedSection
	for _, canonical := range canonicalSections() {
		var items []contribution
		for _, src := range sources {
			if body := src.sections[canonical.Key]; body != "" {
				items = append(items, contribution{Name: src.name, Body: body})
			}
		}
		if len(items) > 0 {
			merged = append(merged, mergedSection{Heading: canonical.Heading, Items: items})
		}
	}
	return merged
}
