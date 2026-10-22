package loader

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

const privateFileName = "pulsarules.yaml"

type commonDirResolver func(dir string) (string, error)

type layers struct {
	base, private *projectfile.File
	preset        string
	rules         projectfile.Rules
}

func (layered *Layered) loadLayers(
	root string, resolve commonDirResolver, presetFlag string,
) (*projectfile.File, error) {
	base, baseErr := loadOptional(filepath.Join(root, projectfile.Name),
		projectfile.Name, layered.rules)
	rules := privateRules(layered.rules, base)
	// A failed base hides its delegated ids, so the private ids cannot be judged yet.
	rules.AnyAnalyzer = baseErr != nil
	private, privateErr := loadPrivate(root, resolve, rules)
	switch {
	case baseErr != nil && privateErr != nil:
		return nil, joinFaults(baseErr, privateErr)
	case baseErr != nil:
		return nil, baseErr
	case privateErr != nil:
		return nil, privateErr
	case private == nil:
		return base, nil
	case base == nil:
		return private, nil
	}
	stack := layers{base: base, private: private, preset: presetFlag, rules: layered.rules}
	return stack.combine()
}

func (stack layers) combine() (*projectfile.File, error) {
	// Each file alone must build, so a fault of one file is never blamed on the merge.
	for _, layer := range []*projectfile.File{stack.base, stack.private} {
		if _, err := Build(layer, ""); err != nil {
			return nil, &fileError{File: layer.Path, Err: err}
		}
	}
	merged, err := stack.merge()
	if err != nil {
		return nil, &fileError{
			File: stack.base.Path, Lead: mergedName(stack.base, stack.private), Err: err,
		}
	}
	if err = stack.checkOverrides(merged); err != nil {
		return nil, err
	}
	return merged, nil
}

func (stack layers) checkOverrides(merged *projectfile.File) error {
	private := stack.private
	// Checked against the merged project state, but numbered as in the private file.
	project := *merged
	project.Overrides = merged.Overrides[:len(merged.Overrides)-len(private.Overrides)]
	cfg, err := Build(&project, stack.preset)
	if err != nil {
		return &fileError{File: faultFile(nil, merged), Err: err}
	}
	if err = private.CheckOverrides(cfg); err != nil {
		return &fileError{File: private.Path, Err: err}
	}
	return nil
}

func loadPrivate(
	root string, resolve commonDirResolver, rules projectfile.Rules,
) (*projectfile.File, error) {
	gitDir, err := resolve(root)
	if errors.Is(err, vcs.ErrNoRepository) || errors.Is(err, vcs.ErrGitNotFound) {
		return nil, nil
	}
	if err != nil {
		// Anything else may hide a private file that should apply, so the gates must not run.
		// The directory is unknown, so the file is named by where it would live.
		expected := "<git-common-dir>/" + privateFileName
		reason := &reworded{text: "locate private config: " + withoutRoot(err, root), cause: err}
		return nil, &fileError{File: expected, Err: reason}
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
