package hook

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

func (d *Dispatcher) emitPreEdit(session *SessionTracker, in hookPayload) error {
	switch strings.ToLower(filepath.Ext(in.ToolInput.FilePath)) {
	case ".go", ".sql":
	default:
		return nil
	}
	if !session.OncePerSession("pre-edit") {
		return nil
	}
	return d.emitContext("hooks/pre-edit.txt", "PreToolUse")
}

func (d *Dispatcher) emitPostEdit(session *SessionTracker, in hookPayload) error {
	filePath := in.ToolInput.FilePath
	relevant := d.router.SkillsForFile(filePath)
	if len(relevant) == 0 {
		return nil
	}

	projectDir := d.resolveProjectDir()
	if projectDir != "" {
		skillsDir := filepath.Join(projectDir, ".claude", "skills")
		relevant = filterInstalled(relevant, skillsDir)
	}
	if len(relevant) == 0 {
		return d.emitContext("hooks/post-edit.txt", "PostToolUse")
	}

	text, err := renderPostEditChecklist(d.templates, postEditChecklist{
		BaseName: filepath.Base(filePath),
		SkillIDs: relevant,
		IsGo:     strings.ToLower(filepath.Ext(filePath)) == ".go",
	})
	if err != nil {
		return err
	}

	if !session.FirstEmission("PostToolUse", text) {
		return nil
	}
	d.emitOutput("PostToolUse", text)
	return nil
}

// postEditChecklist is the data the post-edit-checklist template renders: the
// edited file's base name, the skills matched and installed for it, and
// whether the file is Go (which adds the doc-comment reminder line).
type postEditChecklist struct {
	BaseName string
	SkillIDs []string
	IsGo     bool
}

// renderPostEditChecklist executes the post-edit-checklist template against
// doc, so the agent-facing text can be edited without recompiling, the same
// way every other hook event's text lives in knowledge/templates/hooks/*.txt.
func renderPostEditChecklist(templates fs.FS, doc postEditChecklist) (string, error) {
	asset := "hooks/post-edit-checklist.txt.tmpl"
	body, err := fs.ReadFile(templates, asset)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", asset, err)
	}
	tmpl, err := template.New(asset).Parse(string(body))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", asset, err)
	}
	var buf strings.Builder
	if execErr := tmpl.Execute(&buf, doc); execErr != nil {
		return "", fmt.Errorf("render %s: %w", asset, execErr)
	}
	return strings.TrimSpace(buf.String()), nil
}
