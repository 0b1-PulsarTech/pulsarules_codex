package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/throwsite"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/projectroot"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func (disp *Dispatcher) emitPreEdit(session *sessionTracker, in hookPayload) error {
	filePath := in.ToolInput.FilePath
	if !disp.router.Routes(filePath) {
		return nil
	}
	// Gate per file path, not per session: the failure this hook guards
	// against recurs at each routed file's first write, so muting after file 1
	// would leave files 2..N unguarded. A re-edit of the same file stays silent.
	if !session.FirstEmission(preEditFileEvent(filePath), filePath) {
		return nil
	}
	return disp.emitContext("hooks/pre-edit.txt", "PreToolUse")
}

func preEditFileEvent(filePath string) string { return fileEvent("pre-edit-file-", filePath) }

func fileEvent(prefix, filePath string) string {
	const eventKeyBytes = 16
	sum := sha256.Sum256([]byte(filePath))
	// Half the sha256 sum; plenty to avoid collisions among edited files.
	return prefix + hex.EncodeToString(sum[:eventKeyBytes])
}

func (disp *Dispatcher) emitPostEdit(session *sessionTracker, in hookPayload) error {
	filePath := in.ToolInput.FilePath

	// Before the skills gate below, which returns early for any extension
	// that routes no skill; no trigger names .md, where most markers live.
	notice := joinNotice(disp.autoCleanEdited(filePath), disp.throwNotice(session, filePath))

	if !disp.router.Routes(filePath) {
		if notice == "" {
			return nil
		}
		disp.emitOutput("PostToolUse", notice)
		return nil
	}

	relevant := disp.router.SkillsForFile(filePath)
	if skillsDir := disp.resolveSkillsDir(); skillsDir != "" {
		relevant = filterInstalled(relevant, skillsDir)
	} else {
		// Without a skills dir we cannot tell which of the matched skills
		// are actually installed here, and naming un-installed skills is
		// wrong output, so degrade to the generic reminder instead.
		relevant = nil
	}
	if len(relevant) == 0 {
		// post-edit.txt names Go commands, which mean nothing for a Dart or TS file.
		asset := "hooks/post-edit-other.txt"
		if isGoFile(filePath) {
			asset = "hooks/post-edit.txt"
		}
		return disp.emitContextWithNotice(asset, "PostToolUse", notice)
	}

	text, err := renderPostEditChecklist(disp.templates, postEditChecklist{
		BaseName: printable.Path(filepath.Base(filePath)),
		SkillIDs: relevant,
		IsGo:     isGoFile(filePath),
	})
	if err != nil {
		return err
	}

	// The checklist dedups on content, but a mutation notice must not: a
	// second cleanup of the same file is a new fact, not a repeat.
	if !session.FirstEmission("PostToolUse", text) && notice == "" {
		return nil
	}
	disp.emitOutput("PostToolUse", joinNotice(notice, text))
	return nil
}

func joinNotice(notice, text string) string {
	if notice == "" {
		return text
	}
	if text == "" {
		return notice
	}
	return notice + "\n\n" + text
}

type postEditChecklist struct {
	BaseName string
	SkillIDs []string
	IsGo     bool
}

func isGoFile(filePath string) bool { return strings.ToLower(filepath.Ext(filePath)) == ".go" }

func renderPostEditChecklist(templatesItem fs.FS, doc postEditChecklist) (string, error) {
	return templates.New(templatesItem).Render("hooks/post-edit-checklist.txt.tmpl", doc)
}

func (disp *Dispatcher) installedSkill(id string) string {
	skillsDir := disp.resolveSkillsDir()
	if skillsDir == "" || len(filterInstalled([]string{id}, skillsDir)) == 0 {
		return ""
	}
	return id
}

func (disp *Dispatcher) throwNotice(session *sessionTracker, filePath string) string {
	root := disp.resolveProjectDir()
	ext := filepath.Ext(filePath)
	if root == "" || !throwsite.Handles(ext) {
		return ""
	}
	resolved, err := projectroot.ResolveInside(root, filePath)
	if err != nil {
		disp.debugThrowSkip(filePath, err.Error())
		return ""
	}
	if reason := projectroot.StatGate(resolved); reason != "" {
		disp.debugThrowSkip(filePath, reason)
		return ""
	}
	src, err := os.ReadFile(resolved) //nolint:gosec // resolved by ResolveInside.
	if err != nil {
		disp.debugThrowSkip(filePath, err.Error())
		return ""
	}
	hits := throwsite.Scan(ext, string(src))
	if len(hits) == 0 {
		return ""
	}
	// A skill that is not installed here must not be named, as in the checklist.
	skill := disp.installedSkill(throwsite.SkillFor(ext))
	text := throwsite.Notice(printable.Path(filepath.Base(filePath)), hits, skill)
	// Per file, so editing another file between two edits does not re-announce this one.
	if !session.FirstEmission(fileEvent("post-edit-throw-", filePath), text) {
		return ""
	}
	return text
}

func (disp *Dispatcher) debugThrowSkip(filePath, reason string) {
	if disp.logger != nil {
		disp.logger.Debug("throw scan skipped", "path", filePath, "reason", reason)
	}
}
