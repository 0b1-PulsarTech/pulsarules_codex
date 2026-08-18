package agentswire

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// defaultProjectDescription is used when the installer has no richer description
// for the target project; the author is expected to refine it.
const defaultProjectDescription = "A Go service following the pulsarules_codex engineering standards."

// skillRef is one skill listed in the generated AGENTS.md.
type skillRef struct {
	ID, Description string
}

// simplification: the template's "## Stop signs" section still hardcodes
// nine prohibitions by hand, duplicating `forbidden` clauses that already
// live across knowledge/standards/rules/**, instead of deriving from them
// like the routing contract above now does. Deriving it needs a renderer
// that walks every rule's forbidden section, selects the stop-sign-worthy
// subset, and dedupes wording - a bigger change than unifying the routing
// contract this package renders. Upgrade path: build that renderer (likely
// in internal/skill/render, which already walks rule bodies for clauselint)
// once the two copies are caught disagreeing, or once a second stop-sign
// consumer needs the same list.
type agentsDoc struct {
	ProjectName        string
	ProjectDescription string
	Contract           string
	Skills             []skillRef
}

// WriteAgents renders the AGENTS.md template into <projectDir>/AGENTS.md,
// listing the selected skills (ids) and folding in the mandatory routing
// contract. It is the one builder every install target that wants AGENTS.md
// at the project root calls (opencode's and the thin agents-only layout), so
// the file can never diverge between them: one root AGENTS.md, not one per
// layout. It reports whether it wrote the file: a root AGENTS.md already
// present without marker.Installed is a user's own file - a name they very
// plausibly own already - so WriteAgents leaves it untouched rather than
// clobbering it, the same ownership discipline RemoveAgents applies to
// deletion.
func WriteAgents(
	templates fs.FS, projectDir string, idx *knowledge.Index, ids []string,
) (wrote bool, err error) {
	path := filepath.Join(projectDir, "AGENTS.md")
	var exists, ours bool
	exists, ours, err = marker.Check(path)
	if err != nil {
		return false, fmt.Errorf("check %q: %w", path, err)
	}
	if exists && !ours {
		return false, nil
	}

	var tmpl *template.Template
	tmpl, err = template.ParseFS(templates, "docs/AGENTS.md.tmpl")
	if err != nil {
		return false, fmt.Errorf("parse AGENTS.md.tmpl: %w", err)
	}
	var contractText string
	contractText, err = contract.Session(templates)
	if err != nil {
		return false, fmt.Errorf("routing contract: %w", err)
	}
	doc := agentsDoc{
		ProjectName:        filepath.Base(projectDir),
		ProjectDescription: defaultProjectDescription,
		Contract:           contractText,
		Skills:             skillRefs(idx, ids),
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, doc); err != nil {
		return false, fmt.Errorf("render AGENTS.md: %w", err)
	}

	if err = os.MkdirAll(projectDir, fsperm.DirPrivate); err != nil {
		return false, fmt.Errorf("mkdir %q: %w", projectDir, err)
	}
	//nolint:gosec // path is under the caller's project dir.
	if err = os.WriteFile(path, buf.Bytes(), fsperm.FilePrivate); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}
	return true, nil
}

// skillRefs lists the selected skills (ids) in composition order, each
// reduced to its id and the first sentence of its description. It is scoped
// to ids rather than the whole catalog: AGENTS.md now lives at the project
// root, where a skill listed but never rendered anywhere on disk (the thin
// agents target writes nothing else) would be a dead reference no host could
// follow, and the scoping also keeps the file honest about what a --skills
// subset actually installed.
func skillRefs(idx *knowledge.Index, ids []string) []skillRef {
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}
	ordered := idx.SkillsOrdered()
	refs := make([]skillRef, 0, len(ids))
	for _, skill := range ordered {
		if !selected[skill.ID] {
			continue
		}
		refs = append(refs, skillRef{
			ID: skill.ID, Description: knowledge.FirstSentence(skill.Description),
		})
	}
	return refs
}
