package pathalias

import (
	"encoding/json"
	"maps"
	"path"
	"slices"
	"strings"
)

type compilerFile struct {
	Extends         json.RawMessage `json:"extends"`
	CompilerOptions struct {
		BaseURL *string             `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

type compilerSettings struct {
	baseURL, pathsDir string
	hasBase           bool
	paths             map[string][]string
}

func (finder *Finder) pathAliases(dir string) []Alias {
	var aliases []Alias
	for _, name := range []string{"tsconfig.json", "jsconfig.json"} {
		settings := finder.settings(path.Join(dir, name), 0, map[string]bool{})
		aliases = append(aliases, settings.aliases()...)
	}
	return aliases
}

func (finder *Finder) settings(file string, hop int, seen map[string]bool) compilerSettings {
	const maxExtendsHops = 4
	if hop > maxExtendsHops || seen[file] {
		return compilerSettings{}
	}
	seen[file] = true
	body, ok := finder.read(file)
	if !ok {
		return compilerSettings{}
	}
	stripped, ok := stripJSONC(body)
	var parsed compilerFile
	if !ok || json.Unmarshal(stripped, &parsed) != nil {
		return compilerSettings{}
	}
	dir := path.Dir(file)
	var out compilerSettings
	if parent := extendsFile(dir, parsed.Extends); parent != "" {
		out = finder.settings(parent, hop+1, seen)
	}
	if base := parsed.CompilerOptions.BaseURL; base != nil {
		out.baseURL, out.hasBase = path.Join(dir, *base), true
	}
	if parsed.CompilerOptions.Paths != nil {
		out.paths, out.pathsDir = parsed.CompilerOptions.Paths, dir
	}
	return out
}

func extendsFile(dir string, raw json.RawMessage) string {
	var value string
	if json.Unmarshal(raw, &value) != nil || value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "./") && !strings.HasPrefix(value, "../") {
		return ""
	}
	if !strings.HasSuffix(value, ".json") {
		value += ".json"
	}
	return path.Join(dir, value)
}

func (settings compilerSettings) aliases() []Alias {
	base := settings.pathsDir
	if settings.hasBase {
		base = settings.baseURL
	}
	var found []Alias
	for _, pattern := range slices.Sorted(maps.Keys(settings.paths)) {
		targets := settings.paths[pattern]
		if pattern == "*" || len(targets) == 0 {
			continue
		}
		target := path.Join(base, targets[0])
		if target == ".." || strings.HasPrefix(target, "../") {
			continue
		}
		found = append(found, Alias{Spec: pattern, Target: target, Origin: OriginPaths})
	}
	return found
}
