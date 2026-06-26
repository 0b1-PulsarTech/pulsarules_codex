package render

import (
	"fmt"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// LintSections checks that every rule/pattern body parses as a template and only
// defines canonical section keys (so no section is silently dropped at render).
// It returns one message per problem (empty when clean).
func LintSections(idx *knowledge.Index) []string {
	canonical := map[string]bool{}
	for _, section := range canonicalSections {
		canonical[section.Key] = true
	}
	var problems []string
	for _, host := range allHosts(idx) {
		defines, err := bodyDefines(idx.Body(host.kind, host.id))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s %q: %v", host.kind, host.id, err))
			continue
		}
		for _, name := range defines {
			if !canonical[name] {
				problems = append(problems, fmt.Sprintf(
					"%s %q defines non-canonical section %q", host.kind, host.id, name,
				))
			}
		}
	}
	return problems
}

type host struct{ kind, id string }

func allHosts(idx *knowledge.Index) []host {
	hosts := make([]host, 0, len(idx.Rules)+len(idx.Patterns))
	for _, rule := range idx.Rules {
		hosts = append(hosts, host{"rules", rule.ID})
	}
	for _, pattern := range idx.Patterns {
		hosts = append(hosts, host{"patterns", pattern.ID})
	}
	return hosts
}

// bodyDefines parses a body and returns the names of its `{{define}}` blocks.
func bodyDefines(body string) ([]string, error) {
	parsed, err := template.New("body").Funcs(funcs()).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	var names []string
	for _, assoc := range parsed.Templates() {
		if assoc.Name() != "body" {
			names = append(names, assoc.Name())
		}
	}
	return names, nil
}
