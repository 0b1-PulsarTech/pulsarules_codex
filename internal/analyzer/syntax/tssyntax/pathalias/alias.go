package pathalias

import (
	"slices"
	"strings"
)

type Origin int

const (
	OriginImports Origin = iota
	OriginPaths
	OriginKit
)

type Alias struct {
	Spec, Target, Root string
	Origin             Origin
}

func Match(pattern, spec string) (string, bool) {
	prefix, suffix, wild := strings.Cut(pattern, "*")
	if !wild {
		return "", pattern == spec
	}
	if len(spec) < len(prefix)+len(suffix) ||
		!strings.HasPrefix(spec, prefix) || !strings.HasSuffix(spec, suffix) {
		return "", false
	}
	return spec[len(prefix) : len(spec)-len(suffix)], true
}

func Specs(aliases []Alias) []string {
	var specs []string
	for _, alias := range aliases {
		if alias.Usable() && !slices.Contains(specs, alias.Spec) {
			specs = append(specs, alias.Spec)
		}
	}
	return specs
}

func (alias Alias) Resolve(spec string) (string, bool) {
	star, ok := Match(alias.Spec, spec)
	if !ok {
		return "", false
	}
	return strings.Replace(alias.Target, "*", star, 1), true
}

func (alias Alias) Spelling(target string) (string, bool) {
	if !strings.Contains(alias.Spec, "*") {
		if strings.Contains(alias.Target, "*") || target != alias.Target {
			return "", false
		}
		return alias.Spec, true
	}
	specBase, specOK := strings.CutSuffix(alias.Spec, "*")
	dir, dirOK := alias.dir()
	if !specOK || !dirOK {
		return "", false
	}
	rest, under := Within(dir, target)
	if !under || rest == "" {
		return "", false
	}
	return specBase + rest, true
}

func (alias Alias) dir() (string, bool) {
	if alias.Target == "*" {
		return ".", true
	}
	return strings.CutSuffix(alias.Target, "/*")
}

func Within(dir, target string) (string, bool) {
	if target == dir {
		return "", true
	}
	if dir == "." {
		return target, target != ".." && !strings.HasPrefix(target, "../")
	}
	return strings.CutPrefix(target, dir+"/")
}
