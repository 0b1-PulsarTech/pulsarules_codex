package glob

import "slices"

type Set struct {
	patterns []Pattern
	isOpen   bool
}

func CompileSet(sources []string) Set {
	set := Set{isOpen: len(sources) == 0}
	for _, source := range sources {
		// Config validation rejects a bad glob; one that still fails matches nothing.
		if pattern, err := Compile(source); err == nil {
			set.patterns = append(set.patterns, pattern)
		}
	}
	return set
}

func (set Set) IsOpen() bool { return set.isOpen }

func (set Set) Match(file string) bool {
	if set.isOpen {
		return true
	}
	return slices.ContainsFunc(set.patterns, func(pattern Pattern) bool {
		return pattern.Match(file)
	})
}
