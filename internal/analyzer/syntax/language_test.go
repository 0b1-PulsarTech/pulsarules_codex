package syntax

import "testing"

func TestLanguageForExtension(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext  string
		want Language
	}{
		{ext: ".go", want: LangGo},
		{ext: ".dart", want: LangDart},
		{ext: ".ts", want: LangTypeScript},
		{ext: ".mts", want: ""},
		{ext: ".cts", want: ""},
		{ext: ".tsx", want: LangTypeScript},
		{ext: ".js", want: LangJavaScript},
		{ext: ".mjs", want: LangJavaScript},
		{ext: ".cjs", want: LangJavaScript},
		{ext: ".jsx", want: LangJavaScript},
		{ext: ".svelte", want: LangSvelte},
		{ext: ".GO", want: LangGo},
		{ext: ".py", want: ""},
		{ext: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.ext, func(t *testing.T) {
			t.Parallel()

			if got := LanguageForExtension(testCase.ext); got != testCase.want {
				t.Fatalf("got %q for %q, want %q", got, testCase.ext, testCase.want)
			}
		})
	}
}
