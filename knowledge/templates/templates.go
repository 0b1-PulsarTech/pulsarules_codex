package templates

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

type Set struct {
	fs.FS
}

func New(fsys fs.FS) Set { return Set{FS: fsys} }

func (set Set) Text(name string) (string, error) {
	raw, err := fs.ReadFile(set.FS, name)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func (set Set) Render(name string, input any) (string, error) {
	raw, err := fs.ReadFile(set.FS, name)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	tmpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", name, err)
	}
	var buf strings.Builder
	if execErr := tmpl.Execute(&buf, input); execErr != nil {
		return "", fmt.Errorf("render %s: %w", name, execErr)
	}
	return strings.TrimSpace(buf.String()), nil
}
