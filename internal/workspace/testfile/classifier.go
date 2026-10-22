package testfile

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

type Classifier struct {
	rules    map[language.ID]rule
	fallback rule
	globs    []glob.Pattern
}

func Defaults() Classifier {
	return Classifier{rules: languageRules(), fallback: fallbackRule()}
}

func New(globs []string) (Classifier, error) {
	classifier := Defaults()
	patterns, err := glob.CompileAll(globs)
	if err != nil {
		return Classifier{}, fmt.Errorf("test glob: %w", err)
	}
	classifier.globs = patterns
	return classifier, nil
}

func (classifier Classifier) IsTest(file string) bool {
	slashed := filepath.ToSlash(file)
	if slices.ContainsFunc(classifier.globs, func(pattern glob.Pattern) bool {
		return pattern.Match(slashed)
	}) {
		return true
	}
	return classifier.ruleFor(path.Ext(slashed)).matches(slashed)
}

func (classifier Classifier) ruleFor(ext string) rule {
	rules, fallback := classifier.rules, classifier.fallback
	// The zero value behaves like Defaults, as the zero Skips set does.
	if rules == nil {
		rules, fallback = languageRules(), fallbackRule()
	}
	if selected, found := rules[language.ForExtension(ext).ID]; found {
		return selected
	}
	return fallback
}

func (selected rule) matches(file string) bool {
	// Suffixes match case-sensitively on the original base name, as go test and dart test do.
	if slices.ContainsFunc(selected.suffixes, func(suffix string) bool {
		return strings.HasSuffix(path.Base(file), suffix)
	}) {
		return true
	}
	dir, name := path.Split(strings.ToLower(file))
	if slices.ContainsFunc(selected.markers, func(marker string) bool {
		return strings.Contains(name, marker)
	}) {
		return true
	}
	// SvelteKit route files start with a plus and are never tests by directory.
	if selected.routePlus && strings.HasPrefix(name, "+") {
		return false
	}
	if slices.ContainsFunc(strings.Split(dir, "/"), func(segment string) bool {
		return slices.Contains(selected.dirs, segment)
	}) {
		return true
	}
	// Exact directory names keep their case, as dart test reads them.
	original, _ := path.Split(file)
	return slices.ContainsFunc(strings.Split(original, "/"), func(segment string) bool {
		return slices.Contains(selected.exactDirs, segment)
	})
}
