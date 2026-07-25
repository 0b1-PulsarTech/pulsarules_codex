package hook

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func (d *Dispatcher) emitUserPrompt(session *sessionTracker) error {
	if !session.OncePerSession("prompt") {
		return nil
	}
	return d.emitContext("hooks/user-prompt.txt", "UserPromptSubmit")
}

func (d *Dispatcher) emitStop(event string, session *sessionTracker) (int, error) {
	projectDir := d.resolveProjectDir()
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
	block, count := d.governance(repo, status)
	if block == "" {
		// Nothing actionable: stay silent rather than nagging about a
		// dirty tree the governance checks have nothing to say about.
		return 0, nil
	}
	text, err := fs.ReadFile(d.templates, "hooks/stop.txt")
	if err != nil {
		return count, fmt.Errorf("read stop.txt: %w", err)
	}
	context := strings.TrimSpace(string(text)) + block
	if checklist := TypedChecklist(status); checklist != "" {
		context += "\n" + checklist
	}
	context += "\n\nUncommitted changes:\n" + status.String()
	if !session.FirstEmission(event, context) {
		return count, nil
	}
	d.emitOutput(event, context)
	return count, nil
}
