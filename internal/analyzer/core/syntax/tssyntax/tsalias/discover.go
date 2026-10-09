package tsalias

import (
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
)

type home struct {
	root  string
	found bool
}

type Finder struct {
	read     func(string) ([]byte, bool)
	projects map[string]home
	aliases  map[string][]Alias
}

func NewFinder(read func(string) ([]byte, bool)) *Finder {
	return &Finder{
		read:     read,
		projects: map[string]home{},
		aliases:  map[string][]Alias{},
	}
}

func (finder *Finder) Root(file string) (string, bool) {
	return finder.project(path.Dir(path.Clean(file)))
}

func (finder *Finder) For(file string) []Alias {
	root, ok := finder.Root(file)
	if !ok {
		return nil
	}
	if cached, seen := finder.aliases[root]; seen {
		return cached
	}
	found := finder.discover(root)
	finder.aliases[root] = found
	return found
}

func (finder *Finder) project(dir string) (string, bool) {
	if known, seen := finder.projects[dir]; seen {
		return known.root, known.found
	}
	var known home
	if _, ok := finder.read(path.Join(dir, "package.json")); ok {
		known = home{root: dir, found: true}
	} else if dir != "." && dir != "/" {
		known.root, known.found = finder.project(path.Dir(dir))
	}
	finder.projects[dir] = known
	return known.root, known.found
}

type manifest struct {
	Imports         map[string]json.RawMessage `json:"imports"`
	Dependencies    map[string]json.RawMessage `json:"dependencies"`
	DevDependencies map[string]json.RawMessage `json:"devDependencies"`
}

func (finder *Finder) discover(root string) []Alias {
	const kitPackage = "@sveltejs/kit"
	body, _ := finder.read(path.Join(root, "package.json"))
	var parsed manifest
	if json.Unmarshal(body, &parsed) != nil {
		return nil
	}
	aliases := importAliases(root, parsed.Imports)
	aliases = append(aliases, finder.pathAliases(root)...)
	_, kit := parsed.Dependencies[kitPackage]
	_, kitDev := parsed.DevDependencies[kitPackage]
	if kit || kitDev {
		aliases = append(aliases, kitDefault(root))
		aliases = append(aliases, finder.kitAliases(root)...)
	}
	return unique(aliases)
}

func importAliases(root string, imports map[string]json.RawMessage) []Alias {
	var aliases []Alias
	for _, spec := range slices.Sorted(maps.Keys(imports)) {
		target, ok := firstString(imports[spec])
		if !ok || !strings.HasPrefix(target, "./") || !strings.HasPrefix(spec, "#") {
			continue
		}
		aliases = append(aliases, Alias{Spec: spec, Target: path.Join(root, target)})
	}
	return aliases
}

func kitDefault(root string) Alias {
	return Alias{
		Spec: "$lib/*", Target: path.Join(root, "src/lib") + "/*", Root: root, Origin: OriginKit,
	}
}

func unique(aliases []Alias) []Alias {
	seen := map[string]bool{}
	return slices.DeleteFunc(aliases, func(alias Alias) bool {
		key := fmt.Sprint(alias.Origin, alias.Spec)
		duplicate := seen[key]
		seen[key] = true
		return duplicate
	})
}

func firstString(raw json.RawMessage) (string, bool) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	return nextString(decoder)
}

func nextString(decoder *json.Decoder) (string, bool) {
	token, err := decoder.Token()
	if err != nil {
		return "", false
	}
	switch value := token.(type) {
	case string:
		return value, true
	case json.Delim:
		return firstInContainer(decoder, value)
	}
	return "", false
}

func firstInContainer(decoder *json.Decoder, open json.Delim) (string, bool) {
	for decoder.More() {
		if open == '{' {
			key, err := decoder.Token()
			if err != nil {
				return "", false
			}
			if key == "types" {
				var skipped json.RawMessage
				if decoder.Decode(&skipped) != nil {
					return "", false
				}
				continue
			}
		}
		if found, ok := nextString(decoder); ok {
			return found, true
		}
	}
	// Consume the closing delimiter; a malformed stream fails the next Token call.
	_, _ = decoder.Token()
	return "", false
}
