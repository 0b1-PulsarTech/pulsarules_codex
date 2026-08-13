package render

import (
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

var ErrUnknownComposition = errors.New("unknown composition")

func HasNormativeSection(idx *knowledge.Index, skill knowledge.Skill) (bool, error) {
	sources, err := mergeSources(idx, skill)
	if err != nil {
		return false, fmt.Errorf("skill %q: %w", skill.ID, err)
	}
	for _, key := range normativeSectionKeys() {
		for _, src := range sources {
			if src.sections[key] != "" {
				return true, nil
			}
		}
	}
	return false, nil
}
