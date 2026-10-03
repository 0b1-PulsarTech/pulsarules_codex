package gosrc

import "testing"

func TestReplacePackageWord(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		comment string
		want    string
	}{
		{"word swapped", "// Package util does x.", "// Package routing does x."},
		{
			"longer identifier untouched",
			"// Package utility does x.",
			"// Package utility does x.",
		},
		{"absent", "// does x.", "// does x."},
		{"end of comment", "// Package util", "// Package routing"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := replacePackageWord(
				testCase.comment,
				"util",
				"routing",
			); got != testCase.want {
				t.Fatalf("replacePackageWord() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestPreambleMatches(t *testing.T) {
	t.Parallel()

	const oldSrc = "//go:build linux\n\n// Package util does x.\npackage util\n"
	testCases := []struct {
		name   string
		newSrc string
		want   bool
	}{
		{"same", oldSrc, true},
		{
			"doc follows the package name",
			"//go:build linux\n\n// Package routing does x.\npackage routing\n",
			true,
		},
		{
			"another doc text",
			"//go:build linux\n\n// Package routing does y.\npackage routing\n",
			false,
		},
		{
			"another constraint",
			"//go:build windows\n\n// Package util does x.\npackage util\n",
			false,
		},
		{"constraint dropped", "// Package util does x.\npackage util\n", false},
		{"doc dropped", "//go:build linux\n\npackage util\n", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := PreambleMatches(mustParse(t, oldSrc), mustParse(t, testCase.newSrc))
			if got != testCase.want {
				t.Fatalf("PreambleMatches() = %v, want %v", got, testCase.want)
			}
		})
	}
}
