package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func (disp *Dispatcher) emitPreSearch(session *sessionTracker, in hookPayload) error {
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
	return disp.emitContext("hooks/pre-search.txt", "PreToolUse")
}

func searchTargetsGo(in hookPayload) bool {
	for _, field := range []string{in.ToolInput.Pattern, in.ToolInput.Glob, in.ToolInput.Path} {
		if strings.Contains(strings.ToLower(field), ".go") {
			return true
		}
	}
	if in.ToolName != "Bash" {
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

func (disp *Dispatcher) emitUserPrompt() error {
	return disp.emitContext("hooks/user-prompt.txt", "UserPromptSubmit")
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
	result := disp.governanceWithin(repo, status, entered)
	outcome := stopOutcome{findings: result.count, timedOut: result.timedOut}
	if result.block == "" {
		// Nothing actionable: stay silent rather than nagging about a
		// dirty tree the governance checks have nothing to say about.
		return stopOutcome{}, nil
	}
	context, err := disp.stopHeader(result)
	if err != nil {
		return outcome, err
	}
	bullets, err := disp.stopChecklist(status)
	if err != nil {
		return outcome, err
	}
	if bullets != "" {
		context += "\n" + bullets
	}
	context += "\n\nUncommitted changes:\n" + cappedStatus(status)
	if !session.FirstEmission(event, context) {
		return outcome, nil
	}
	disp.emitOutput(event, context)
	return outcome, nil
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
		return status.String()
	}
	shown := vcs.Status{Changes: status.Changes[:maxListedChanges]}
	return fmt.Sprintf("%s\n... and %d more", shown.String(), len(status.Changes)-maxListedChanges)
}
