package detect

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type Language = language.ID

func All() []Language {
	return []Language{
		language.Go, language.Dart, language.JavaScript, language.TypeScript,
		language.Svelte, language.Rust, language.Python,
	}
}

func ParseLanguage(raw string) (Language, error) {
	lang := Language(strings.ToLower(strings.TrimSpace(raw)))
	if !slices.Contains(All(), lang) {
		return "", fmt.Errorf("unknown language %q (want one of %s)", raw, joinLanguages(All()))
	}
	return lang, nil
}

func ParseNames(names []string) (Set, error) {
	langs := make([]Language, 0, len(names))
	for _, name := range names {
		lang, err := ParseLanguage(name)
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
