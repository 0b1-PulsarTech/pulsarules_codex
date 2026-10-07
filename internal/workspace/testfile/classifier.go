package testfile

import (
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type Classifier struct {
	rules    map[syntax.Language]rule
	fallback rule
}

func Defaults() Classifier {
	return Classifier{rules: languageRules(), fallback: fallbackRule()}
}

func (policy Classifier) IsTest(file string) bool {
	slashed := filepath.ToSlash(file)
	return policy.ruleFor(path.Ext(slashed)).matches(slashed)
}

func (policy Classifier) ruleFor(ext string) rule {
	rules, fallback := policy.rules, policy.fallback
	// The zero value behaves like Defaults, as the zero Skips set does.
	if rules == nil {
		rules, fallback = languageRules(), fallbackRule()
	}
	if selected, found := rules[syntax.LanguageForExtension(ext)]; found {
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
