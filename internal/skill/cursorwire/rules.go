package cursorwire

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// RulesDir is where Cursor project rules live, relative to the project root:
// one flat .mdc file per rule, unlike the nested <id>/SKILL.md + .gitignore
// convention output.WriteDoc uses for Claude and opencode. Cursor has no
// per-id directory, so ownership of a file is proven the same way
// agentswire proves it owns AGENTS.md: a marker comment baked into the
// rendered body (the "mdcFrontmatter" partial in
// knowledge/templates/skills/parts.tmpl), not a sibling file.
const RulesDir = ".cursor/rules"

// PointerID is the filename (minus .mdc) of the always-on pointer rule: the
// one file every Cursor request sees, so its content is deliberately kept
// small - see cursor.go's buildPointerBody for the size rationale.
const PointerID = "pulsarules-contract"

// WriteRule writes body (a rendered .mdc rule, frontmatter and all) to
// <projectDir>/.cursor/rules/<id>.mdc. It refuses to overwrite a file at that
// path that lacks marker.Installed - a user-authored rule this tool never
// wrote - mirroring agentswire.WriteAgents' ownership check, and reports
// whether it wrote the file.
func WriteRule(projectDir, id, body string) (wrote bool, err error) {
	path := filepath.Join(projectDir, RulesDir, id+".mdc")
	var ours bool
	ours, err = ownsExisting(path)
	if err != nil {
		return false, err
	}
	if !ours {
		return false, nil
	}
	if err = os.MkdirAll(filepath.Dir(path), fsperm.DirPrivate); err != nil {
		return false, fmt.Errorf("mkdir: %w", err)
	}
	//nolint:gosec // path is under the caller's project dir.
	if err = os.WriteFile(path, []byte(body), fsperm.FilePrivate); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}
	return true, nil
}

// ownsExisting reports whether path may be written: true when nothing is
// there yet, or when the existing content carries marker.Installed (a file
// this tool wrote before, safe to refresh).
func ownsExisting(path string) (bool, error) {
	exists, ours, err := marker.Check(path)
	if err != nil {
		return false, fmt.Errorf("check %q: %w", path, err)
	}
	return !exists || ours, nil
}
