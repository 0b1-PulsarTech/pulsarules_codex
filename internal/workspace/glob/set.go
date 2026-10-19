package glob

import "slices"

type Set struct {
	patterns     []Pattern
	isRestricted bool
}

func CompileSet(sources []string) Set {
	set := Set{isRestricted: len(sources) > 0}
	for _, source := range sources {
		// Config validation rejects a bad glob; one that still fails matches nothing.
		if pattern, err := Compile(source); err == nil {
			set.patterns = append(set.patterns, pattern)
		}
	}
	return set
}

func CompileAll(sources []string) ([]Pattern, error) {
	patterns := make([]Pattern, 0, len(sources))
	for _, source := range sources {
		pattern, err := Compile(source)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

func (set Set) IsOpen() bool { return !set.isRestricted }

func (set Set) Match(file string) bool {
	if !set.isRestricted {
		return true
	}
	return slices.ContainsFunc(set.patterns, func(pattern Pattern) bool {
		return pattern.Match(file)
	})
}
