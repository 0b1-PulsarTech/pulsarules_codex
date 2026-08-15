package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// canonicalSections lists the generic section keys in render order with the
// heading each merges under. A rule/pattern body declares the ones it has as
// `{{define "<key>"}}` blocks; the controller merges same-keyed sections across a
// skill's composed rules and patterns.
var canonicalSections = []struct{ Key, Heading string }{
	{"when", "When this applies"},
	{"catalog", "The catalog"},
	{"must", "Must"},
	{"recipe", "Recipe"},
	{"approved", "Approved patterns and where they appear"},
	{"rejected", "Rejected anti-patterns (and the rule each breaks)"},
	{"examples", "Examples"},
	{"forbidden", "Forbidden"},
	{"validation", "Validation checklist"},
	// outputs renders last: the pre-refactor sidecars carried "Expected outputs"
	// after Forbidden actions too, since it summarizes the payoff of everything
	// above rather than gating it.
	{"outputs", "Expected outputs"},
}

// contribution is one rule's or pattern's text for a merged section.
type contribution struct {
	Name string
	Body string
}

// mergedSection is one canonical section with every composed source's text.
type mergedSection struct {
	Heading string
	Items   []contribution
}

// bodySections parses a body in its own namespace (so bare `{{define "must"}}`
// names do not collide across rules) and returns each canonical section it
// defines, trimmed.
func bodySections(body string) (map[string]string, error) {
	parsed, err := template.New("body").Funcs(funcs()).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	sections := map[string]string{}
	for _, canonical := range canonicalSections {
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

// mergeSources resolves a skill's composed rules and patterns to named sources in
// composition order, each carrying its parsed sections.
func mergeSources(idx *knowledge.Index, skill knowledge.Skill) ([]source, error) {
	var sources []source
	add := func(kind, entry, name string) error {
		id, _, _ := strings.Cut(entry, "#")
		sections, err := bodySections(idx.Body(kind, id))
		if err != nil {
			return fmt.Errorf("%s %q: %w", kind, id, err)
		}
		sources = append(sources, source{name: name, sections: sections})
		return nil
	}
	for _, entry := range skill.ComposeRules {
		id, _, _ := strings.Cut(entry, "#")
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
		id, _, _ := strings.Cut(entry, "#")
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

// normativeSectionKeys are the composed section keys that state an obligation
// (what a skill's consumer must, must not, or must check) rather than merely
// describe or demonstrate one.
var normativeSectionKeys = []string{"must", "forbidden", "validation"}
