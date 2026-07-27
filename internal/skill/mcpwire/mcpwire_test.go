package mcpwire

import (
	"testing/fstest"
)

func fakeTemplates() fstest.MapFS {
	return fstest.MapFS{
		"mcp/mcp.json.tmpl": {Data: []byte(`{
  "mcpServers": {
    {{- range $i, $s := .Servers }}
    {{- if $i }},{{ end }}
    "{{ $s.Name }}": {
      "command": "{{ $s.Command }}",
      "args": [ {{- range $j, $a := $s.Args }}{{ if $j }}, {{ end }}"{{ $a }}"{{- end }} ],
      "cwd": "{{ $.RepoDir }}"
    }
    {{- end }}
  }
}
`)},
	}
}
