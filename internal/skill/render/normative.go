package render

import (
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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
