package gosrc

import (
	"go/parser"
	"go/token"
	"slices"
	"testing"
)

func TestEmbedPatterns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want []string
	}{
		{"none", "package p\n\n// go:embed notes\nvar V int\n", nil},
		{"one literal", embedSrc("//go:embed public.txt"), []string{"public.txt"}},
		{
			"several on a line and a second directive",
			"package p\n\n//go:embed a.txt  b/*.txt\n//go:embed c.txt\nvar V string\n",
			[]string{"a.txt", "b/*.txt", "c.txt"},
		},
		{
			"quoted with a space",
			embedSrc("//go:embed \"my file.txt\" `raw.txt`"),
			[]string{"my file.txt", "raw.txt"},
		},
		{"unterminated quote kept raw", embedSrc("//go:embed \"broken"), []string{"\"broken"}},
		{"a longer directive name is not an embed", embedSrc("//go:embedded x"), nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(
				token.NewFileSet(),
				"",
				testCase.src,
				parser.ParseComments,
			)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := embedPatterns(file); !slices.Equal(got, testCase.want) {
				t.Fatalf("embedPatterns() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func embedSrc(directive string) string {
	return "package p\n\nimport _ \"embed\"\n\n" + directive + "\nvar Data string\n"
}
