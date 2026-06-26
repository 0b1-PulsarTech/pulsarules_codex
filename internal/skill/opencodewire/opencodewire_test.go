package opencodewire

import (
	"testing/fstest"
)

// fakeTemplates is the minimal templates filesystem the opencodewire tests render
// against: the AGENTS.md template with the fields WriteAgents fills.
func fakeTemplates() fstest.MapFS {
	return fstest.MapFS{
		"docs/AGENTS.md.tmpl": {Data: []byte(`# AGENTS.md - {{.ProjectName}}

{{.ProjectDescription}}

Skills live under {{.SkillsDir}}/.

## Available skills
{{- range .Skills}}
- ` + "`{{.ID}}`" + ` - {{.Description}}
{{- end}}
`)},
	}
}
