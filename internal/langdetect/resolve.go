package langdetect

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func Resolve(repoDir, overrideCSV string) (Set, error) {
	if overrideCSV != "" {
		set, err := ParseCSV(overrideCSV)
		if err != nil {
			return Set{}, fmt.Errorf("parse language override: %w", err)
		}
		if err = SaveConfig(repoDir, set); err != nil {
			return Set{}, fmt.Errorf("persist language override: %w", err)
		}
		return set, nil
	}

	set, ok, err := LoadConfig(repoDir)
	if err != nil {
		return Set{}, fmt.Errorf("load language config: %w", err)
	}
	if ok {
		return set, nil
	}

	// Markers live at depth 0 to 2 in a real layout; the cap leaves one level of slack above.
	set, err = Detect(repoDir, core.SkipSet{}, DefaultMaxDepth)
	if err != nil {
		return Set{}, fmt.Errorf("detect languages: %w", err)
	}
	// An empty detection is a non-answer: markers not written yet, or an
	// install pointed below them. Recording it would freeze that non-answer,
	// since a later run trusts the config instead of re-detecting.
	if set.Empty() {
		return set, nil
	}
	if err = SaveConfig(repoDir, set); err != nil {
		return Set{}, fmt.Errorf("persist detected languages: %w", err)
	}
	return set, nil
}
