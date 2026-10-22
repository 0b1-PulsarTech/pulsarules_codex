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

const FileName = "AGENTS.md"

func Write(
	templates fs.FS, projectDir string, idx *knowledge.Index, ids []string,
) (ownedfile.Outcome, error) {
	tmpl, err := template.ParseFS(templates, "docs/AGENTS.md.tmpl")
	if err != nil {
		return ownedfile.Kept, fmt.Errorf("parse AGENTS.md.tmpl: %w", err)
	}
	contractText, err := contract.Session(templates)
	if err != nil {
		return ownedfile.Kept, fmt.Errorf("routing contract: %w", err)
	}
	const defaultProjectDescription = "A Go service following the pulsarules_codex " +
		"engineering standards."
	doc := agentsDoc{
		ProjectName:        filepath.Base(projectDir),
		ProjectDescription: defaultProjectDescription,
		Contract:           contractText,
		Skills:             skillRefs(idx, ids),
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, doc); err != nil {
		return ownedfile.Kept, fmt.Errorf("render AGENTS.md: %w", err)
	}
	path := filepath.Join(projectDir, FileName)
	return ownedfile.WriteOwned(path, buf.Bytes(), disk.ModeFilePrivate, makeProjectDir)
}

func makeProjectDir(dir string) error {
	if err := os.MkdirAll(dir, disk.ModeDirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	return nil
}

func skillRefs(idx *knowledge.Index, ids []string) []skillRef {
	// Scope to ids so AGENTS.md never lists a skill that was not installed.
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
