package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

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

func bodySections(body string) (map[string]string, error) {
	// Each body parses in its own namespace so bare {{define "must"}} names do not collide.
	parsed, err := template.New("body").Funcs(funcs()).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	sections := map[string]string{}
	for _, canonical := range canonicalSections() {
		if parsed.Lookup(canonical.Key) == nil {
			continue
		}
		var buf bytes.Buffer
		if execErr := parsed.ExecuteTemplate(&buf, canonical.Key, nil); execErr != nil {
			return nil, fmt.Errorf("render section %q: %w", canonical.Key, execErr)
		}
		sections[canonical.Key] = strings.TrimSpace(buf.String())
	}
	return sections, nil
}

func mergeSources(idx *knowledge.Index, skill knowledge.Skill) ([]source, error) {
	var sources []source
	add := func(kind, entry, name string) error {
		id := entryID(entry)
		sections, err := bodySections(idx.Body(kind, id))
		if err != nil {
			return fmt.Errorf("%s %q: %w", kind, id, err)
		}
		sources = append(sources, source{name: name, sections: sections})
		return nil
	}
	for _, entry := range skill.ComposeRules {
		id := entryID(entry)
		rule, ok := idx.Rule(id)
		if !ok {
			return nil, fmt.Errorf(
				"%w: skill %q composes unknown rule %q", ErrUnknownComposition, skill.ID, id,
			)
		}
		if err := add("rules", entry, rule.Name); err != nil {
			return nil, err
		}
	}
	for _, entry := range skill.ComposePatterns {
		id := entryID(entry)
		pattern, ok := idx.Pattern(id)
		if !ok {
			return nil, fmt.Errorf(
				"%w: skill %q composes unknown pattern %q", ErrUnknownComposition, skill.ID, id,
			)
		}
		if err := add("patterns", entry, pattern.Name); err != nil {
			return nil, err
		}
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
