package hook

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/throwsite"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/projectroot"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func (disp *Dispatcher) emitPreEdit(session *sessionTracker, in hookPayload, host string) error {
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
	return disp.emitContext("hooks/pre-edit.txt", host)
}

func preEditFileEvent(filePath string) string { return fileEvent("pre-edit-file-", filePath) }

func fileEvent(prefix, filePath string) string {
	const eventKeyHexChars = 32
	// Half the sha256 sum; plenty to avoid collisions among edited files.
	return prefix + fingerprint([]byte(filePath))[:eventKeyHexChars]
}

func (disp *Dispatcher) emitPostEdit(session *sessionTracker, in hookPayload, host string) error {
	filePath := in.ToolInput.FilePath

	// Before the skills gate below, which returns early for any extension
	// that routes no skill; no trigger names .md, where most markers live.
	notice := joinNotice(disp.autoCleanEdited(filePath), disp.throwNotice(session, filePath))

	if !disp.router.Routes(filePath) {
		if notice == "" {
			return nil
		}
		disp.emitOutput(host, notice)
		return nil
	}

	// Without a skills dir no skill counts as installed, and naming an un-installed
	// skill is wrong output, so the result degrades to the generic reminder.
	relevant := slices.DeleteFunc(disp.router.SkillsForFile(filePath), func(id string) bool {
		return !disp.skillInstalled(id)
	})
	if len(relevant) == 0 {
		// post-edit.txt names Go commands, which mean nothing for a Dart or TS file.
		asset := "hooks/post-edit-other.txt"
		if isGoFile(filePath) {
			asset = "hooks/post-edit.txt"
		}
		return disp.emitContextWithNotice(asset, host, notice)
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
	if !session.FirstEmission(host, text) && notice == "" {
		return nil
	}
	disp.emitOutput(host, joinNotice(notice, text))
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

func isGoFile(filePath string) bool {
	return language.ForExtension(filepath.Ext(filePath)).ID == language.Go
}

func renderPostEditChecklist(assets templates.Set, doc postEditChecklist) (string, error) {
	return assets.Render("hooks/post-edit-checklist.txt.tmpl", doc)
}

func (disp *Dispatcher) throwNotice(session *sessionTracker, filePath string) string {
	root := disp.projectDir
	ext := filepath.Ext(filePath)
	if root == "" || !disp.throws.Handles(ext) {
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
	hits := disp.throws.Scan(ext, string(src))
	if len(hits) == 0 {
		return ""
	}
	// A skill that is not installed here must not be named, as in the checklist.
	skill := disp.throws.SkillFor(ext)
	if !disp.skillInstalled(skill) {
		skill = ""
	}
	text := throwsite.Notice(printable.Path(filepath.Base(filePath)), hits, skill)
	// Per file, so editing another file between two edits does not re-announce this one.
	if !session.FirstEmission(fileEvent("post-edit-throw-", filePath), text) {
		return ""
	}
	return text
}

func (disp *Dispatcher) debugThrowSkip(filePath, reason string) {
	disp.logger.Debug("throw scan skipped", "path", filePath, "reason", reason)
}
