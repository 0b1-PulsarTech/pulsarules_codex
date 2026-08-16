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

	// Before the skills gate below, which returns early for any extension
	// that routes no skill; no trigger names .md, where most markers live.
	notice := d.autoCleanEdited(filePath)

	relevant := d.router.SkillsForFile(filePath)
	if len(relevant) == 0 {
		if notice == "" {
			return nil
		}
		d.emitOutput("PostToolUse", notice)
		return nil
	}

	if skillsDir := d.resolveSkillsDir(); skillsDir != "" {
		relevant = filterInstalled(relevant, skillsDir)
	} else {
		// Without a skills dir we cannot tell which of the matched skills
		// are actually installed here, and naming un-installed skills is
		// wrong output, so degrade to the generic reminder instead.
		relevant = nil
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

	// The checklist dedups on content, but a mutation notice must not: a
	// second cleanup of the same file is a new fact, not a repeat.
	if !session.FirstEmission("PostToolUse", text) && notice == "" {
		return nil
	}
	d.emitOutput("PostToolUse", joinNotice(notice, text))
	return nil
}

func joinNotice(notice, text string) string {
	if notice == "" {
		return text
	}
	return notice + "\n\n" + text
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
