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

// why: ask the router instead of carrying a second hardcoded extension list.
// post-edit already asks Router.SkillsForFile for the same filePath; a
// second hand-maintained list can silently drift from it (it had: .proto
// matches grpc-adapter's trigger and got the post-edit checklist, but this
// switch never included ".proto" so pre-edit stayed silent for it).
func (d *Dispatcher) emitPreEdit(session *SessionTracker, in hookPayload) error {
	filePath := in.ToolInput.FilePath
	if len(d.router.SkillsForFile(filePath)) == 0 {
		return nil
	}
	// why: gate per file path, not once per session - the measured failure
	// this hook exists for (router runs, then skips loading matched skills
	// at write time) recurs at EACH routed file's first write, so muting
	// after file 1 leaves files 2..N unguarded. FirstEmission's content-hash
	// then keeps a re-edit of the SAME file silent (the marker key is file-specific).
	if !session.FirstEmission(preEditFileEvent(filePath), filePath) {
		return nil
	}
	return d.emitContext("hooks/pre-edit.txt", "PreToolUse")
}

// why: a raw file path contains "/", which markerPath cannot embed as a
// single marker filename component (os.WriteFile would try to create it as a
// nested path with no parent directory and fail). Hashing produces a stable,
// filesystem-safe token instead; it need not be reversible, only distinct
// per distinct path and stable across repeat calls for the same path.
func preEditFileEvent(filePath string) string {
	const eventKeyBytes = 16 // half the sha256 sum; plenty to avoid collisions among edited files
	sum := sha256.Sum256([]byte(filePath))
	return "pre-edit-file-" + hex.EncodeToString(sum[:eventKeyBytes])
}

func (d *Dispatcher) emitPostEdit(session *SessionTracker, in hookPayload) error {
	filePath := in.ToolInput.FilePath

	// why: before the skills gate below, which returns early for any extension
	// that routes no skill - and no trigger names .md, where most markers live.
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
		// wrong output - degrade to the generic reminder instead.
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

	// why: the checklist dedups on content, but a mutation notice must not - a
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
