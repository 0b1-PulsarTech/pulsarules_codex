package loader

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

const (
	privateFileName = "pulsarules.yaml"
	userDirName     = "pulsarules"
)

type commonDirResolver func(dir string) (string, error)

type layers struct {
	base, private *projectfile.File
	preset        string
	rules         projectfile.Rules
}

type layerLoader func(rules projectfile.Rules) (*projectfile.File, error)

func (layered *Layered) loadLayers(
	root string, resolve commonDirResolver, presetFlag string,
) (*projectfile.File, error) {
	loaders := []layerLoader{
		func(rules projectfile.Rules) (*projectfile.File, error) {
			return layered.loadUser(root, rules)
		},
		func(rules projectfile.Rules) (*projectfile.File, error) {
			return loadOptional(filepath.Join(root, projectfile.Name), projectfile.Name, rules)
		},
		func(rules projectfile.Rules) (*projectfile.File, error) {
			return loadPrivate(root, resolve, rules)
		},
	}
	rules := layered.rules
	var user *projectfile.File
	var files []*projectfile.File
	var faults []error
	for at, load := range loaders {
		file, err := load(rules)
		if err != nil {
			faults = append(faults, err)
			// A failed layer hides its delegated ids, so the layers above cannot be judged yet.
			rules.AnyAnalyzer = true
			continue
		}
		if file == nil {
			continue
		}
		rules = privateRules(rules, file)
		if at == 0 {
			user = file
			continue
		}
		files = append(files, file)
	}
	if len(faults) > 0 {
		return nil, joinAll(faults)
	}
	return layered.stack(user, files, presetFlag)
}

func joinAll(faults []error) error {
	joined := faults[0]
	for _, next := range faults[1:] {
		joined = joinFaults(joined, next)
	}
	return joined
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
