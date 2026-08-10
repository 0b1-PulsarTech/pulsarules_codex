package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

func (d *Dispatcher) emitPreEdit(session *SessionTracker, in hookPayload) error {
	filePath := in.ToolInput.FilePath
	if len(d.router.SkillsForFile(filePath)) == 0 {
		return nil
	}
	// Gate per file path, not per session: the failure this hook guards
	// against recurs at each routed file's first write, so muting after file 1
	// would leave files 2..N unguarded. A re-edit of the same file stays silent.
	if !session.FirstEmission(preEditFileEvent(filePath), filePath) {
		return nil
	}
	return d.emitContext("hooks/pre-edit.txt", "PreToolUse")
}

func preEditFileEvent(filePath string) string {
	const eventKeyBytes = 16
	sum := sha256.Sum256([]byte(filePath))
	// Half the sha256 sum; plenty to avoid collisions among edited files.
	return "pre-edit-file-" + hex.EncodeToString(sum[:eventKeyBytes])
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

type postEditChecklist struct {
	BaseName string
	SkillIDs []string
	IsGo     bool
}

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
