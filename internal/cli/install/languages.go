package install

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

func resolveLanguages(cfg installConfig, override []string) (detect.Set, error) {
	if cfg.global {
		return globalLanguages(override)
	}
	languages, err := detect.Resolve(cfg.absBase, override)
	if err != nil {
		return detect.Set{}, fmt.Errorf("resolve project languages: %w", err)
	}
	if languages.Empty() {
		// An empty set installs every skill, not none.
		_, _ = fmt.Fprintf(os.Stderr,
			"warning: no project language detected in %q; installing every skill "+
				"regardless of language. Pass --languages to say which apply.\n", cfg.absBase)
	}
	return languages, nil
}

func globalLanguages(override []string) (detect.Set, error) {
	if override == nil {
		// A global install belongs to no project: nothing is scanned under the home directory.
		return detect.NewSet(detect.All()...), nil
	}
	set, err := detect.ParseNames(override)
	if err != nil {
		return detect.Set{}, fmt.Errorf("parse language override: %w", err)
	}
	return set, nil
}
