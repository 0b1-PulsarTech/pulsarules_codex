package language

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestLookup_UnknownNameReturnsNil(t *testing.T) {
	t.Parallel()

	if got := Lookup("cobol"); got != nil {
		t.Errorf("Lookup(%q) = %v, want nil", "cobol", got)
	}
}

func TestLookup_Grammar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		languageName string
		wantExts     []string
	}{
		{"go", "go", []string{".go"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			language := Lookup(testCase.languageName)
			if language == nil {
				t.Fatalf("Lookup(%q) = nil, want a handler", testCase.languageName)
			}
			exts := language.Extensions()
			for _, ext := range testCase.wantExts {
				if !slices.Contains(exts, ext) {
					t.Errorf("Extensions() = %v, want it to contain %q", exts, ext)
				}
			}
		})
	}
}

func TestRegisterAll_RegistersEveryExtension(t *testing.T) {
	t.Parallel()

	reg := pass.NewLanguageRegistry()
	RegisterAll(reg)

	testCases := []string{".go"}
	for _, ext := range testCases {
		if lang := reg.Lookup(ext); lang == nil {
			t.Errorf("registry lookup for %q returned nil after RegisterAll", ext)
		}
	}
}
