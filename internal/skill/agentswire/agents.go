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
		return false, fmt.Errorf("render AGENTS.md: %w", err)
	}

	if err = os.MkdirAll(projectDir, fsperm.DirPrivate); err != nil {
		return false, fmt.Errorf("mkdir %q: %w", projectDir, err)
	}
	// Path is under the caller's project dir.
	if err = os.WriteFile(path, buf.Bytes(), fsperm.FilePrivate); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}
	return true, nil
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
