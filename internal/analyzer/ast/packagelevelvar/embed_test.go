package packagelevelvar

import "testing"

func TestAnalyzeEmbed(t *testing.T) {
	t.Parallel()

	const hdr = "package p\n\nimport \"embed\"\n\n"
	runCases(t, []levelVarCase{
		{
			name:  "embed directive on decl exempt",
			files: map[string]string{"a.go": hdr + "//go:embed assets\nvar assets embed.FS\n"},
		},
		{
			name:  "embed directive then blank line exempt",
			files: map[string]string{"a.go": hdr + "//go:embed assets\n\nvar assets embed.FS\n"},
		},
		{
			name: "embed directive then comment and blank lines exempt",
			files: map[string]string{
				"a.go": hdr + "//go:embed assets\n// note\n\nvar assets embed.FS\n",
			},
		},
		{
			name: "embed directive on spec in group exempt",
			files: map[string]string{
				"a.go": hdr + "var (\n\t//go:embed a.txt\n\ta string\n\tfs embed.FS\n)\n",
			},
			want: []string{"a.go:8:package-level var fs"},
		},
		{
			name: "group doc embed does not exempt a parenthesized group",
			files: map[string]string{
				"a.go": hdr + "//go:embed a.txt\nvar (\n\ta string\n\tb string\n)\n",
			},
			want: []string{"a.go:7:package-level var a", "a.go:8:package-level var b"},
		},
		{
			name: "directive for an earlier var does not exempt the next",
			files: map[string]string{
				"a.go": hdr + "//go:embed a.txt\nvar a string\n\nvar b string\n",
			},
			want: []string{"a.go:8:package-level var b"},
		},
	})
}
