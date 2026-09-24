package langdetect

import (
	"fmt"
	"slices"
	"strings"
)

type Language string

const (
	Go         Language = "go"
	Dart       Language = "dart"
	JavaScript Language = "javascript"
	TypeScript Language = "typescript"
	Svelte     Language = "svelte"
	Rust       Language = "rust"
	Python     Language = "python"
)

func All() []Language {
	return []Language{Go, Dart, JavaScript, TypeScript, Svelte, Rust, Python}
}

func ParseLanguage(raw string) (Language, error) {
	lang := Language(strings.ToLower(strings.TrimSpace(raw)))
	if !slices.Contains(All(), lang) {
		return "", fmt.Errorf("unknown language %q (want one of %s)", raw, joinLanguages(All()))
	}
	return lang, nil
}

func ParseCSV(csv string) (Set, error) {
	var langs []Language
	for token := range strings.SplitSeq(csv, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		lang, err := ParseLanguage(token)
		if err != nil {
			return Set{}, err
		}
		langs = append(langs, lang)
	}
	return NewSet(langs...), nil
}

func joinLanguages(languages []Language) string {
	names := make([]string, len(languages))
	for idx, lang := range languages {
		names[idx] = string(lang)
	}
	return strings.Join(names, ", ")
}
