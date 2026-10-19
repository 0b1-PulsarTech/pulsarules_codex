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
