package projectcfg

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

const privateFileName = "pulsarules.yaml"

type commonDirResolver func(dir string) (string, error)

func loadLayers(
	root string, resolve commonDirResolver, presetFlag string,
) (*config.ProjectFile, error) {
	base, baseErr := loadOptional(filepath.Join(root, config.ProjectFileName),
		config.ProjectFileName, Rules())
	rules := privateRules(base)
	// A failed base hides its delegated ids, so the private ids cannot be judged yet.
	rules.AnyAnalyzer = baseErr != nil
	private, privateErr := loadPrivate(root, resolve, rules)
	switch {
	case baseErr != nil && privateErr != nil:
		return nil, fmt.Errorf("%w; %w", baseErr, privateErr)
	case baseErr != nil:
		return nil, baseErr
	case privateErr != nil:
		return nil, privateErr
	case private == nil:
		return base, nil
	case base == nil:
		return private, nil
	}
	return mergeLayers(base, private, presetFlag)
}

func mergeLayers(
	base, private *config.ProjectFile, presetFlag string,
) (*config.ProjectFile, error) {
	// Each file alone must build, so a fault of one file is never blamed on the merge.
	for _, layer := range []*config.ProjectFile{base, private} {
		if _, err := Build(layer, ""); err != nil {
			return nil, err
		}
	}
	merged, err := merge(base, private, presetFlag)
	if err != nil {
		return nil, err
	}
	if err = checkPrivateOverrides(merged, private, presetFlag); err != nil {
		return nil, err
	}
	return merged, nil
}

func checkPrivateOverrides(merged, private *config.ProjectFile, presetFlag string) error {
	// Checked against the merged project state, but numbered as in the private file.
	project := *merged
	project.Overrides = merged.Overrides[:len(merged.Overrides)-len(private.Overrides)]
	cfg, err := Build(&project, presetFlag)
	if err != nil {
		return err
	}
	return cfg.CheckOverrides(private)
}

func loadPrivate(
	root string, resolve commonDirResolver, rules config.Rules,
) (*config.ProjectFile, error) {
	gitDir, err := resolve(root)
	if errors.Is(err, vcs.ErrNoRepository) || errors.Is(err, vcs.ErrGitNotFound) {
		return nil, nil
	}
	if err != nil {
		// Anything else may hide a private file that should apply, so the gates must not run.
		return nil, &reworded{text: "locate private config: " + withoutRoot(err, root), cause: err}
	}
	path := filepath.Join(gitDir, privateFileName)
	return loadOptional(path, displayName(root, path), rules)
}

func displayName(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(rel)
}
