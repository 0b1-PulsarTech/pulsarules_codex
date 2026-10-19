package language

import "testing"

func TestLanguages_EveryExtensionHasOneOwner(t *testing.T) {
	t.Parallel()

	owner := map[string]ID{}
	ids := map[ID]bool{}
	for _, lang := range languages() {
		if ids[lang.ID] {
			t.Errorf("language %q is listed twice", lang.ID)
		}
		ids[lang.ID] = true
		if len(lang.extensions) == 0 {
			t.Errorf("language %q has no extension", lang.ID)
		}
		if len(lang.Comments.Line) == 0 && len(lang.Comments.Blocks) == 0 {
			t.Errorf("language %q has no comment grammar", lang.ID)
		}
		for _, ext := range lang.extensions {
			if previous, taken := owner[ext]; taken {
				t.Errorf("extension %q is listed for %q and %q", ext, previous, lang.ID)
			}
			owner[ext] = lang.ID
		}
	}
}

func TestLanguages_LineStringsAndRegions(t *testing.T) {
	t.Parallel()

	for _, lang := range languages() {
		quoting := len(lang.lineStrings.Delims) > 0
		if quoting == (lang.ID == HTML) {
			t.Errorf("language %q: has line strings = %v", lang.ID, quoting)
		}
		if got, want := lang.regions != nil, lang.ID == Svelte || lang.ID == Vue; got != want {
			t.Errorf("language %q: has regions = %v, want %v", lang.ID, got, want)
		}
	}
}
