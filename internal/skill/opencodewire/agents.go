package opencodewire

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// SkillsSubdir is where rendered skills live in an opencode project, relative to
// the project root. It is also the glob root wired into opencode.json instructions.
const SkillsSubdir = ".opencode/skills"

// defaultProjectDescription is used when the installer has no richer description
// for the target project; the author is expected to refine it.
const defaultProjectDescription = "A Go service following the pulsarules_codex engineering standards."

// skillRef is one skill listed in the generated AGENTS.md.
type skillRef struct {
	ID, Description string
}

type agentsDoc struct {
	ProjectName        string
	ProjectDescription string
	SkillsDir          string
	Skills             []skillRef
}

// WriteAgents renders the AGENTS.md template into <projectDir>/.opencode/AGENTS.md,
// listing every skill and folding in the mandatory routing contract. It writes
// under .opencode so it never clobbers a human-authored root AGENTS.md.
func WriteAgents(templates fs.FS, projectDir string, idx *knowledge.Index) error {
	tmpl, err := template.ParseFS(templates, "docs/AGENTS.md.tmpl")
	if err != nil {
		return fmt.Errorf("parse AGENTS.md.tmpl: %w", err)
	}
	doc := agentsDoc{
		ProjectName:        filepath.Base(projectDir),
		ProjectDescription: defaultProjectDescription,
		SkillsDir:          SkillsSubdir,
		Skills:             skillRefs(idx),
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, doc); err != nil {
		return fmt.Errorf("render AGENTS.md: %w", err)
	}

	dir := filepath.Join(projectDir, ".opencode")
	if err = os.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	path := filepath.Join(dir, "AGENTS.md")
	//nolint:gosec // path is under the caller's project dir.
	if err = os.WriteFile(path, buf.Bytes(), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

// skillRefs lists every non-router skill in composition order, plus the router
// first, each reduced to its id and the first sentence of its description.
func skillRefs(idx *knowledge.Index) []skillRef {
	ordered := idx.SkillsOrdered()
	refs := make([]skillRef, 0, len(ordered))
	for _, skill := range ordered {
		refs = append(refs, skillRef{
			ID: skill.ID, Description: knowledge.FirstSentence(skill.Description),
		})
	}
	return refs
}
