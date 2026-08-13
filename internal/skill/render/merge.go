package render

import (
	"bytes"
	"errors"
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

// ErrUnknownComposition marks the one mergeSources failure that is NOT a defect in a body: a skill
// naming a rule or pattern id that does not exist. Callers that already report unresolved
// references separately match on it so they can keep reporting every OTHER failure - notably a
// text/template parse error inside a {{define}} block, which nothing else would catch.
var ErrUnknownComposition = errors.New("unknown composition")

// HasNormativeSection reports whether skill renders at least one non-empty
// "must", "forbidden", or "validation" section across its composed rules and
// patterns. A skill with none is documentation only - it states no obligation
// a caller can be held to.
func HasNormativeSection(idx *knowledge.Index, skill knowledge.Skill) (bool, error) {
	sources, err := mergeSources(idx, skill)
	if err != nil {
		return false, fmt.Errorf("skill %q: %w", skill.ID, err)
	}
	for _, key := range normativeSectionKeys {
		for _, src := range sources {
			if src.sections[key] != "" {
				return true, nil
			}
		}
	}
	return false, nil
}

type source struct {
	name     string
	sections map[string]string
}

// mergeSections groups the sources' sections under one heading per canonical key,
// keeping each source's contribution as a `### Name` subheading.
func mergeSections(sources []source) []mergedSection {
	var merged []mergedSection
	for _, canonical := range canonicalSections {
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

type sectionDoc struct {
	Heading string
	Body    string
}

type sourceDoc struct {
	Name     string
	Sections []sectionDoc
}

// sourceDocs renders each source's own sections in canonical order, for the
// non-merged (opt-out) layout where every rule keeps its sections together.
func sourceDocs(sources []source) []sourceDoc {
	docs := make([]sourceDoc, 0, len(sources))
	for _, src := range sources {
		var sections []sectionDoc
		for _, canonical := range canonicalSections {
			if body := src.sections[canonical.Key]; body != "" {
				sections = append(sections, sectionDoc{Heading: canonical.Heading, Body: body})
			}
		}
		docs = append(docs, sourceDoc{Name: src.name, Sections: sections})
	}
	return docs
}
