package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"
)

//go:embed docs hooks skills
var embedded embed.FS

type Set struct {
	fsys fs.FS
}

func New(fsys fs.FS) Set { return Set{fsys: fsys} }

func Embedded() Set { return Set{fsys: embedded} }

func Dir(path string) Set { return Set{fsys: os.DirFS(path)} }

func (set Set) FS() fs.FS { return set.fsys }

func (set Set) Text(name string) (string, error) {
	raw, err := fs.ReadFile(set.fsys, name)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func (set Set) Render(name string, input any) (string, error) {
	raw, err := fs.ReadFile(set.fsys, name)
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
