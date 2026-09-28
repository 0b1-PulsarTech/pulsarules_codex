package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func fakeOnPath(tb testing.TB, names ...string) {
	tb.Helper()
	dir := tb.TempDir()
	for _, name := range names {
		if err := os.WriteFile(
			filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o700,
		); err != nil {
			tb.Fatalf("write fake %q on PATH: %v", name, err)
		}
	}
	tb.Setenv("PATH", dir)
}

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
		"skills/gopls-navigation.header.md": {
			Data: []byte("---\nname: gopls-navigation\n---\n\n# gopls navigation\n"),
		},
	}
}
