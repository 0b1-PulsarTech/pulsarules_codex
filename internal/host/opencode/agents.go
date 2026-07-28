package opencode

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type skillRef struct {
	ID, Description string
}

// simplification: "## Stop signs" hardcodes ten prohibitions instead of deriving
// them from `forbidden` clauses, since that needs a renderer walking every rule's
// forbidden section. Upgrade path: build that renderer once the two copies disagree.
type agentsDoc struct {
	ProjectName        string
	ProjectDescription string
	SkillsDir          string
	Skills             []skillRef
}

func WriteAgents(templates fs.FS, projectDir string, idx *knowledge.Index) error {
	tmpl, err := template.ParseFS(templates, "docs/AGENTS.md.tmpl")
	if err != nil {
		return fmt.Errorf("parse AGENTS.md.tmpl: %w", err)
	}
	const defaultProjectDescription = "A Go service following the pulsarules_codex engineering " +
		"standards."
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
	if err = os.MkdirAll(dir, disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	path := filepath.Join(dir, "AGENTS.md")
	// Path is under the caller's project dir.
	if err = os.WriteFile(path, buf.Bytes(), disk.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

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
