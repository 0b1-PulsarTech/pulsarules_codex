package agentswire

import "testing/fstest"

func fakeTemplates() fstest.MapFS {
	return fstest.MapFS{
		"docs/AGENTS.md.tmpl": {Data: []byte(`# AGENTS.md - {{.ProjectName}}

<!-- Installed by pulsarules_cli -->

{{.ProjectDescription}}

## Mandatory routing contract

{{.Contract}}

## Available skills
{{- range .Skills}}
- ` + "`{{.ID}}`" + ` - {{.Description}}
{{- end}}
`)},
		"hooks/contract.txt":      {Data: []byte("routing contract text.\n")},
		"hooks/contract-tail.txt": {Data: []byte("commit tail text.\n")},
	}
}
