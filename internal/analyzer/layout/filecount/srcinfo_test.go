package filecount

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

func TestInspectSource(t *testing.T) {
	t.Parallel()

	const entSchema = "package s\n\nimport \"entgo.io/ent\"\n\ntype T struct{ ent.Schema }\n"
	testCases := []struct {
		name string
		src  string
		want srcInfo
	}{
		{"plain", "package s\n", srcInfo{pkg: "s"}},
		{"ent schema", entSchema, srcInfo{pkg: "s", ent: true}},
		{"syntax error in a body keeps the package", "package s\nfunc (\n", srcInfo{pkg: "s"}},
		{"no package clause", "import \"x\"\n", srcInfo{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, _ := parser.ParseFile(
				token.NewFileSet(), "s.go", testCase.src, parser.ParseComments,
			)
			got := inspectSource(gomod.File{Path: "s.go", Syntax: parsed})
			if got != testCase.want {
				t.Fatalf("inspectSource = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestInspectSourceNoSyntax(t *testing.T) {
	t.Parallel()

	if got := inspectSource(gomod.File{Path: "s.go"}); got != (srcInfo{}) {
		t.Fatalf("inspectSource = %+v, want zero", got)
	}
}

func TestAnalyzePackageScope(t *testing.T) {
	t.Parallel()

	runFileCountCases(t, []fileCountCase{
		{
			name: "build ignore file not counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/gen.go": "//go:build ignore\n\npackage main\n",
			}),
			changed: []string{"pkg/f00.go"},
		},
		{
			name: "legacy build ignore file not counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/gen.go": "// +build ignore\n\npackage p\n",
			}),
			changed: []string{"pkg/f00.go"},
		},
		{
			name: "file with another package name not counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/doc.go": "package other\n",
			}),
			changed: []string{"pkg/f00.go"},
		},
		{
			name: "unparsable file still counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/bad.go": "import \"x\"\n",
			}),
			changed: []string{"pkg/f00.go"},
			want:    []string{"pkg/f00.go:1:package pkg has 11 non-test source files, max 10"},
		},
		{
			name: "package tie resolves to the same count",
			files: merged(numbered("pkg", 6), map[string]string{
				"pkg/a1.go": "package a\n", "pkg/a2.go": "package a\n", "pkg/a3.go": "package a\n",
				"pkg/a4.go": "package a\n", "pkg/a5.go": "package a\n", "pkg/a6.go": "package a\n",
			}),
			changed: []string{"pkg/f00.go"},
			params:  map[string]any{"max_files": 5},
			want:    []string{"pkg/f00.go:1:package pkg has 6 non-test source files, max 5"},
		},
	})
}
