package opencodewire

import (
	"testing/fstest"
)

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
