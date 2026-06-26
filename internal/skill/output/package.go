package output

import (
	"archive/zip"
	"fmt"
	"os"
	"path"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// Package renders every skill in-memory and zips it into outPath, one entry per
// skill as "<id>/SKILL.md". A failure mid-loop closes the writer and archive,
// then removes the partial file so a failed run never leaves a corrupt zip
// behind at outPath.
func Package(idx *knowledge.Index, rnd *render.Renderer, outPath string) (err error) {
	if mkdirErr := os.MkdirAll(path.Dir(outPath), fsperm.DirPrivate); mkdirErr != nil {
		return fmt.Errorf("mkdir: %w", mkdirErr)
	}
	//nolint:gosec // outPath is a caller-supplied output destination.
	archive, createErr := os.Create(outPath)
	if createErr != nil {
		return fmt.Errorf("create %q: %w", outPath, createErr)
	}

	writer := zip.NewWriter(archive)
	defer func() {
		if closeErr := writer.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close zip: %w", closeErr)
		}
		if closeErr := archive.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close archive: %w", closeErr)
		}
		if err != nil {
			_ = os.Remove(outPath)
		}
	}()

	for _, skill := range idx.SkillsOrdered() {
		body, renderErr := rnd.RenderSkill(idx, skill, nil)
		if renderErr != nil {
			return fmt.Errorf("render %q: %w", skill.ID, renderErr)
		}
		entry, entryErr := writer.Create(path.Join(skill.ID, "SKILL.md"))
		if entryErr != nil {
			return fmt.Errorf("create entry %q: %w", skill.ID, entryErr)
		}
		if _, writeErr := entry.Write([]byte(body)); writeErr != nil {
			return fmt.Errorf("write %q: %w", skill.ID, writeErr)
		}
	}
	return nil
}
