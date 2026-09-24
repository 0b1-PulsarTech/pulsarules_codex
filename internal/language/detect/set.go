package detect

import "slices"

type Set struct {
	langs map[Language]struct{}
}

func NewSet(languages ...Language) Set {
	langs := make(map[Language]struct{}, len(languages))
	for _, lang := range languages {
		langs[lang] = struct{}{}
	}
	return Set{langs: langs}
}

func (set Set) Has(lang Language) bool {
	_, ok := set.langs[lang]
	return ok
}

func (set Set) Empty() bool { return len(set.langs) == 0 }

func (set Set) Sorted() []Language {
	out := make([]Language, 0, len(set.langs))
	for lang := range set.langs {
		out = append(out, lang)
	}
	// A rendered config or MCP server list never reorders between runs.
	slices.Sort(out)
	return out
}

func (set Set) Strings() []string {
	sorted := set.Sorted()
	out := make([]string, len(sorted))
	for idx, lang := range sorted {
		out[idx] = string(lang)
	}
	return out
}
