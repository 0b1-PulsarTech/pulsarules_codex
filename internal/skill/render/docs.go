package render

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// The types below are the view models the templates in knowledge/templates
// range over. They carry no behaviour: each field exists because a template
// names it, so a template edit and a field here move together. ref and
// referenceDoc are shared with compose.go; the rest are built by Renderer.

type ref struct {
	Name string
	Body string
}

type referenceDoc struct {
	Title, URL, Note string
}

type skillDoc struct {
	ID, Name, Description  string
	Triggers               []string
	Sidecar                string
	Merged                 []mergedSection
	Sources                []sourceDoc
	Workflows              []ref
	References             []referenceDoc
	AlwaysLoad             bool
	Order                  int
	ComposeSkills, Linters []string
}

type skillSummary struct {
	ID, Description string
}

type routerDoc struct {
	ID, Name, Description string
	Baseline              knowledge.RouterBaseline
	Dispatch              []knowledge.RouterDispatchRow
	Order                 []knowledge.RouterOrderStep
	AvailableSkills       []skillSummary
	ShowTestCallout       bool
}

type workflowDoc struct {
	ID, Name, Description string
	Steps                 []string
	Body                  string
	Rules, Patterns       []ref
}
