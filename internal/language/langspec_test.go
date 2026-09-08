package language

import "testing"

func TestHandler_IsPackageDeclarationDefault(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
	}{
		{"empty line", ""},
		{"a line that looks like Go's package clause", "package foo"},
		{"an import line", "import 'package:foo/foo.dart';"},
	}
	for _, lang := range []string{"clike", "hash", "lua", "markup", "sfc"} {
		for _, testCase := range testCases {
			t.Run(lang+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()
				language := Lookup(lang)
				if got := language.IsPackageDeclaration(testCase.line); got {
					t.Errorf(
						"%s: IsPackageDeclaration(%q) = true, want false", lang, testCase.line,
					)
				}
			})
		}
	}
}
