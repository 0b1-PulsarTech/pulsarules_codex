package agentswire

import "testing/fstest"

// fakeTemplates is the minimal templates filesystem the agentswire tests
// render against: the AGENTS.md template with the fields WriteAgents fills,
// including the ownership marker RemoveAgents checks for, plus the two
// contract assets WriteAgents reads to populate {{.Contract}}.
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
