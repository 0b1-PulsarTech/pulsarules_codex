package render

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
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
	// The catalog loads on first use and is shared by every later emojiAnchors call.
	loadCatalog := sync.OnceValues(emoji.NewCatalog)
	return template.FuncMap{
		"inc": func(value int) int { return value + 1 },
		"emojiAnchors": func() (string, error) {
			catalog, err := loadCatalog()
			if err != nil {
				// An error must fail the render rather than produce a wrong commits skill.
				return "", fmt.Errorf("render emoji anchors: %w", err)
			}
			return anchorsText(catalog.Anchors()), nil
		},
	}
}

func anchorsText(anchors []emoji.Anchor) string {
	areas := make([]string, len(anchors))
	for anchorIdx, anchor := range anchors {
		shortcodes := make([]string, len(anchor.Shortcodes))
		for shortcodeIdx, shortcode := range anchor.Shortcodes {
			shortcodes[shortcodeIdx] = "`:" + shortcode + ":`"
		}
		areas[anchorIdx] = anchor.Area + " (" + strings.Join(shortcodes, ", ") + ")"
	}
	return strings.Join(areas, ", ")
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
