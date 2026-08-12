package render

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

func topTemplates() []string {
	return []string{
		"SKILL.md.tmpl", "WORKFLOW.md.tmpl", "router.md.tmpl",
		"SKILL.mdc.tmpl", "router.mdc.tmpl",
	}
}

func funcs() template.FuncMap {
	return template.FuncMap{
		"inc":          func(value int) int { return value + 1 },
		"emojiAnchors": emojiAnchorsText,
	}
}

func emojiAnchorsText() (string, error) {
	anchors, err := emoji.Anchors()
	if err != nil {
		// An error must fail the render rather than produce a wrong commits skill.
		return "", fmt.Errorf("render emoji anchors: %w", err)
	}
	areas := make([]string, len(anchors))
	for anchorIdx, anchor := range anchors {
		shortcodes := make([]string, len(anchor.Shortcodes))
		for shortcodeIdx, shortcode := range anchor.Shortcodes {
			shortcodes[shortcodeIdx] = "`:" + shortcode + ":`"
		}
		areas[anchorIdx] = anchor.Area + " (" + strings.Join(shortcodes, ", ") + ")"
	}
	return strings.Join(areas, ", "), nil
}

func buildNamespace(templates fs.FS) (*template.Template, error) {
	base := template.New("base").Option("missingkey=error").Funcs(funcs())
	if _, err := base.ParseFS(templates, "skills/parts.tmpl"); err != nil {
		return nil, fmt.Errorf("parse parts.tmpl: %w", err)
	}
	for _, name := range topTemplates() {
		if _, err := base.ParseFS(templates, "skills/"+name); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
	}
	return base, nil
}
