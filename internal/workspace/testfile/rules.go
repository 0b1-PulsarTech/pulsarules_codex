package testfile

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"

type rule struct {
	suffixes  []string
	markers   []string
	dirs      []string
	exactDirs []string
	routePlus bool
}

func languageRules() map[syntax.Language]rule {
	web := rule{
		markers:   []string{".test.", ".spec."},
		dirs:      []string{"test", "tests", "__tests__"},
		routePlus: true,
	}
	// A SvelteKit route directory may be named test, so only the file markers count.
	svelte := rule{markers: web.markers}
	return map[syntax.Language]rule{
		syntax.LangGo: {suffixes: []string{"_test.go"}},
		syntax.LangDart: {
			suffixes:  []string{"_test.dart"},
			exactDirs: []string{"test", "integration_test"},
		},
		syntax.LangTypeScript: web,
		syntax.LangJavaScript: web,
		syntax.LangSvelte:     svelte,
	}
}

func fallbackRule() rule {
	return rule{markers: []string{"_test.", ".test.", "_spec.", ".spec."}}
}
