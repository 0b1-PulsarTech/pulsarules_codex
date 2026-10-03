package hook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/checklist"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func (disp *Dispatcher) emitPreSearch(session *SessionTracker, in hookPayload) error {
	if !searchTargetsGo(in) {
		return nil
	}
	projectDir := disp.resolveProjectDir()
	if projectDir == "" {
		return nil
	}
	// The projectDir is under PULSARULES_PROJECT_DIR, a hook-provided project root.
	if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); err != nil {
		return nil //nolint:nilerr // no go.mod means "not a Go project", not a failure.
	}
	skillsDir := disp.resolveSkillsDir()
	if skillsDir == "" || len(filterInstalled([]string{"gopls-navigation"}, skillsDir)) == 0 {
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

func (disp *Dispatcher) emitStop(event string, session *SessionTracker) (int, error) {
	projectDir := disp.resolveProjectDir()
	if projectDir == "" {
		return 0, nil
	}
	repo, err := vcs.Open(projectDir)
	if err != nil {
		// Not a repository, or a real git failure: either way the hook
		// stays quiet rather than nagging on every turn.
		return 0, nil //nolint:nilerr // deliberate: see comment above.
	}
	status, err := repo.WorktreeStatus()
	if err != nil {
		return 0, fmt.Errorf("read worktree status: %w", err)
	}
	if status.IsClean() {
		return 0, nil
	}
	block, count := disp.governance(repo, status)
	if block == "" {
		// Nothing actionable: stay silent rather than nagging about a
		// dirty tree the governance checks have nothing to say about.
		return 0, nil
	}
	text, err := fs.ReadFile(disp.templates, "hooks/stop.txt")
	if err != nil {
		return count, fmt.Errorf("read stop.txt: %w", err)
	}
	context := strings.TrimSpace(string(text)) + block
	if bullets := checklist.Typed(status); bullets != "" {
		context += "\n" + bullets
	}
	context += "\n\nUncommitted changes:\n" + cappedStatus(status)
	if !session.FirstEmission(event, context) {
		return count, nil
	}
	disp.emitOutput(event, context)
	return count, nil
}

func cappedStatus(status vcs.Status) string {
	const maxListedChanges = 50
	if len(status.Changes) <= maxListedChanges {
		return status.String()
	}
	shown := vcs.Status{Changes: status.Changes[:maxListedChanges]}
	return fmt.Sprintf("%s\n... and %d more", shown.String(), len(status.Changes)-maxListedChanges)
}
