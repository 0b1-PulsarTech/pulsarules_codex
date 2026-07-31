package render

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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
