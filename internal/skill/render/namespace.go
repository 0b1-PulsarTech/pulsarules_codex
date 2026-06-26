package render

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

// topTemplates are the executable entry points parsed into the shared namespace.
var topTemplates = []string{"SKILL.md.tmpl", "WORKFLOW.md.tmpl", "router.md.tmpl"}

// funcs is the shared template FuncMap; every namespace member sees it.
func funcs() template.FuncMap {
	return template.FuncMap{
		"inc":          func(value int) int { return value + 1 },
		"emojiAnchors": emojiAnchorsText,
	}
}

// emojiAnchorsText renders the emoji package's area -> family table as the
// prose the commits skill's emoji-selection guidance quotes, so that guidance
// renders from emoji.Anchors instead of a hand-copied list drifting from it.
// A returned error fails the template render: a broken anchors table must
// never render a silently wrong commits skill.
func emojiAnchorsText() (string, error) {
	anchors, err := emoji.Anchors()
	if err != nil {
		return "", fmt.Errorf("render emoji anchors: %w", err)
	}
	areas := make([]string, len(anchors))
	for i, anchor := range anchors {
		shortcodes := make([]string, len(anchor.Shortcodes))
		for j, shortcode := range anchor.Shortcodes {
			shortcodes[j] = "`:" + shortcode + ":`"
		}
		areas[i] = anchor.Area + " (" + strings.Join(shortcodes, ", ") + ")"
	}
	return strings.Join(areas, ", "), nil
}

// buildNamespace parses the shared partials and the top-level templates into one
// namespace with missingkey=error. Rule/pattern bodies are parsed separately
// (one namespace each) by the merge controller, since their bare `{{define}}`
// section names intentionally repeat across files.
func buildNamespace(templates fs.FS) (*template.Template, error) {
	base := template.New("base").Option("missingkey=error").Funcs(funcs())
	if _, err := base.ParseFS(templates, "skills/parts.tmpl"); err != nil {
		return nil, fmt.Errorf("parse parts.tmpl: %w", err)
	}
	for _, name := range topTemplates {
		if _, err := base.ParseFS(templates, "skills/"+name); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
	}
	return base, nil
}
