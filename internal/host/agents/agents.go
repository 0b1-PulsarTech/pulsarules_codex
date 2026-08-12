package agents

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/contract"
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
	Contract           string
	Skills             []skillRef
}

func Write(
	templates fs.FS, projectDir string, idx *knowledge.Index,
) (wrote bool, err error) {
	path := filepath.Join(projectDir, "AGENTS.md")
	var exists, ours bool
	exists, ours, err = ownedfile.Check(path)
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
	const defaultProjectDescription = "A Go service following the pulsarules_codex " +
		"engineering standards."
	doc := agentsDoc{
		ProjectName:        filepath.Base(projectDir),
		ProjectDescription: defaultProjectDescription,
		Contract:           contractText,
		Skills:             skillRefs(idx),
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, doc); err != nil {
		return false, fmt.Errorf("render AGENTS.md: %w", err)
	}

	if err = os.MkdirAll(projectDir, disk.DirPrivate); err != nil {
		return false, fmt.Errorf("mkdir %q: %w", projectDir, err)
	}
	// Path is under the caller's project dir.
	if err = os.WriteFile(path, buf.Bytes(), disk.FilePrivate); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}
	return true, nil
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
