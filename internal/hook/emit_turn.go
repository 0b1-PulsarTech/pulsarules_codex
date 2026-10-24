package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/dialect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func (disp *Dispatcher) emitPreSearch(session *sessionTracker, in hookPayload, host string) error {
	if !searchTargetsGo(in) {
		return nil
	}
	projectDir := disp.projectDir
	if projectDir == "" {
		return nil
	}
	// The projectDir is the hook-provided project root, read from PULSARULES_PROJECT_DIR once.
	if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); err != nil {
		return nil //nolint:nilerr // no go.mod means "not a Go project", not a failure.
	}
	if !disp.skillInstalled("gopls-navigation") {
		return nil
	}
	if !session.OncePerSession("pre-search") {
		return nil
	}
	return disp.emitContext("hooks/pre-search.txt", host)
}

func searchTargetsGo(in hookPayload) bool {
	for _, field := range []string{in.ToolInput.Pattern, in.ToolInput.Glob, in.ToolInput.Path} {
		if strings.Contains(strings.ToLower(field), ".go") {
			return true
		}
	}
	// Codex names its shell tool exec_command; both carry the command text.
	if in.ToolName != "Bash" && in.ToolName != "exec_command" {
		return false
	}
	fields := strings.Fields(in.ToolInput.Command)
	if len(fields) == 0 {
		return false
	}
	switch filepath.Base(fields[0]) {
	case "grep", "rg", "ag", "find":
		return true
	default:
		return false
	}
}

func (disp *Dispatcher) emitUserPrompt(host string) error {
	return disp.emitContext("hooks/user-prompt.txt", host)
}

func (disp *Dispatcher) emitStop(event string, session *sessionTracker) (stopOutcome, error) {
	entered := time.Now()
	projectDir := disp.projectDir
	if projectDir == "" {
		return stopOutcome{}, nil
	}
	repo, err := disp.open(projectDir)
	if err != nil {
		// Not a repository, or a real git failure: either way the hook
		// stays quiet rather than nagging on every turn.
		return stopOutcome{}, nil //nolint:nilerr // deliberate: see comment above.
	}
	status, err := repo.WorktreeStatus()
	if err != nil {
		return stopOutcome{}, fmt.Errorf("read worktree status: %w", err)
	}
	if status.IsClean() {
		return stopOutcome{}, nil
	}
	result := disp.governanceWithin(repo.Root(), status, entered)
	outcome := stopOutcome{findings: result.count, timedOut: result.timedOut}
	if result.block == "" {
		// Nothing actionable: stay silent rather than nagging about a
		// dirty tree the governance checks have nothing to say about.
		return stopOutcome{}, nil
	}
	header, err := disp.stopHeader(result)
	if err != nil {
		return outcome, err
	}
	context := header + disp.stopBody(status)
	// Codex blocks on the findings alone, so the same findings plus an unrelated file change
	// must not block again; Claude only adds context and keeps the full text as its key.
	dedupeKey := context
	if disp.dialect == dialect.Codex {
		dedupeKey = header
	}
	if !session.FirstEmission(event, dedupeKey) {
		return outcome, nil
	}
	disp.emitOutput(event, context)
	return outcome, nil
}

func (disp *Dispatcher) stopBody(status vcs.Status) string {
	bullets, err := disp.stopChecklist(status)
	if err != nil {
		// The block is the point of the Stop hook; a missing checklist only drops the bullets.
		disp.logger.Warn("stop checklist unavailable", "error", err.Error())
	}
	body := ""
	if bullets != "" {
		body = "\n" + bullets
	}
	return body + "\n\nUncommitted changes:\n" + cappedStatus(status)
}

type stopChecklistFlags struct {
	Go, SQL, YAML, Markdown bool
}

func (disp *Dispatcher) stopChecklist(status vcs.Status) (string, error) {
	present := status.Extensions()
	text, err := disp.templates.Render("hooks/stop-checklist.txt.tmpl", stopChecklistFlags{
		Go: present[".go"], SQL: present[".sql"], YAML: present[".yaml"], Markdown: present[".md"],
	})
	if err != nil || text == "" {
		return "", err
	}
	// Render trims the template, so the bullet indent is added here.
	return "  " + strings.ReplaceAll(text, "\n", "\n  "), nil
}

func (disp *Dispatcher) stopHeader(result governanceResult) (string, error) {
	if result.timedOut {
		return strings.TrimSpace(result.block), nil
	}
	text, err := disp.templates.Text("hooks/stop.txt")
	if err != nil {
		return "", err
	}
	return text + result.block, nil
}

func cappedStatus(status vcs.Status) string {
	const maxListedChanges = 50
	if len(status.Changes) <= maxListedChanges {
		return formatChanges(status.Changes)
	}
	return fmt.Sprintf(
		"%s\n... and %d more",
		formatChanges(status.Changes[:maxListedChanges]), len(status.Changes)-maxListedChanges,
	)
}

func formatChanges(changes []vcs.Change) string {
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		flags := string([]byte{change.Staging, change.Worktree})
		if change.OldPath != "" {
			lines = append(lines, fmt.Sprintf(
				"%s %s -> %s", flags, printable.Path(change.OldPath), printable.Path(change.Path),
			))
			continue
		}
		lines = append(lines, flags+" "+printable.Path(change.Path))
	}
	return strings.Join(lines, "\n")
}
