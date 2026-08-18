package hook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func (d *Dispatcher) emitPreSearch(session *SessionTracker, in hookPayload) error {
	if !searchTargetsGo(in) {
		return nil
	}
	projectDir := d.resolveProjectDir()
	if projectDir == "" {
		return nil
	}
	//nolint:gosec // path is under PULSARULES_PROJECT_DIR, a hook-provided project root.
	if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); err != nil {
		return nil
	}
	skillsDir := d.resolveSkillsDir()
	if skillsDir == "" || len(filterInstalled([]string{"gopls-navigation"}, skillsDir)) == 0 {
		return nil
	}
	if !session.OncePerSession("pre-search") {
		return nil
	}
	return d.emitContext("hooks/pre-search.txt", "PreToolUse")
}

// searchTargetsGo reports whether in describes a Grep/Glob search whose
// pattern/glob/path names a .go file, or a Bash command invoking a
// text-search tool (grep/rg/ag/find) - the two shapes worth nudging toward
// the gopls MCP instead of textual search.
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

// why: no OncePerSession gate - the digest must fire every turn, since
// session drift back off the router happens on turn twenty, not turn one.
func (d *Dispatcher) emitUserPrompt() error {
	return d.emitContext("hooks/user-prompt.txt", "UserPromptSubmit")
}

func (d *Dispatcher) emitStop(event string, session *SessionTracker) (int, error) {
	projectDir := d.resolveProjectDir()
	if projectDir == "" {
		return 0, nil
	}
	repo, err := vcs.Open(projectDir)
	if err != nil {
		// Not a repository, or a real git failure: either way the hook
		// stays quiet rather than nagging on every turn.
		return 0, nil
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
