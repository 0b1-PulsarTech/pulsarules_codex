package opencode

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

const SkillsSubdir = ".opencode/skills"

func mergeSkills(doc *jsondoc.Document) error {
	skills, err := doc.Child("skills")
	if err != nil {
		return err
	}
	// Wire skills.paths, the lazy index opencode expands only on a tool call; never
	// instructions, which opencode reads in full every session.
	paths, err := jsondoc.Section[[]string](skills, "paths")
	if err != nil {
		return err
	}
	if !slices.Contains(paths, SkillsSubdir) {
		paths = append(paths, SkillsSubdir)
	}
	if err = jsondoc.SetSection(skills, "paths", paths); err != nil {
		return err
	}
	return doc.PutChild("skills", skills)
}
