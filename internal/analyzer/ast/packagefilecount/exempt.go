package packagefilecount

import (
	"fmt"
	"path"
	"strings"
)

type exemptions struct {
	patterns []string
}

func (exempt exemptions) matches(dir string) bool {
	for _, pattern := range exempt.patterns {
		if base, isTree := strings.CutSuffix(pattern, "/**"); isTree {
			if matchesAncestor(base, dir) {
				return true
			}
			continue
		}
		if ok, err := path.Match(pattern, dir); err == nil && ok {
			return true
		}
	}
	return false
}

func (exempt exemptions) invalid() []string {
	var problems []string
	for _, pattern := range exempt.patterns {
		probe, _ := strings.CutSuffix(pattern, "/**")
		if _, err := path.Match(probe, ""); err != nil {
			problems = append(problems, fmt.Sprintf("invalid exempt pattern %q: %v", pattern, err))
		}
	}
	return problems
}

func matchesAncestor(base, dir string) bool {
	for prefix := dir; ; prefix = path.Dir(prefix) {
		if ok, err := path.Match(base, prefix); err == nil && ok {
			return true
		}
		if prefix == "." || prefix == "/" {
			return false
		}
	}
}
