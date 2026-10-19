package testfile

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/language"

type rule struct {
	suffixes  []string
	markers   []string
	dirs      []string
	exactDirs []string
	routePlus bool
}

func languageRules() map[language.ID]rule {
	web := rule{
		markers:   []string{".test.", ".spec."},
		dirs:      []string{"test", "tests", "__tests__"},
		routePlus: true,
	}
	// A SvelteKit route directory may be named test, so only the file markers count.
	svelte := rule{markers: web.markers}
	return map[language.ID]rule{
		language.Go: {suffixes: []string{"_test.go"}},
		language.Dart: {
			suffixes:  []string{"_test.dart"},
			exactDirs: []string{"test", "integration_test"},
		},
		language.TypeScript: web,
		language.JavaScript: web,
		language.Svelte:     svelte,
	}
}

func fallbackRule() rule {
	return rule{markers: []string{"_test.", ".test.", "_spec.", ".spec."}}
}
