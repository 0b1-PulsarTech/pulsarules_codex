package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func Generate(idx *knowledge.Index, rnd *render.Renderer, outDir string) ([]string, error) {
	skillsDir := filepath.Join(outDir, "skills")
	if err := os.MkdirAll(skillsDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir %q: %w", skillsDir, err)
	}
	written := make([]string, 0, len(idx.Skills))
	for _, skill := range idx.SkillsOrdered() {
		body, renderErr := rnd.RenderSkill(idx, skill, nil)
		if renderErr != nil {
			return nil, fmt.Errorf("render %q: %w", skill.ID, renderErr)
		}
		skillPath := filepath.Join(skillsDir, skill.ID, "SKILL.md")
		if writeErr := writeFile(skillPath, body); writeErr != nil {
			return nil, fmt.Errorf("write %q: %w", skillPath, writeErr)
		}
		written = append(written, skill.ID)
	}
	return written, nil
}
