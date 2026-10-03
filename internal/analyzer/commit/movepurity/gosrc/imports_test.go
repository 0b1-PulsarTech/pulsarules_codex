package gosrc

import (
	"testing"
)

func TestLocalName(t *testing.T) {
	t.Parallel()

	imports := []ImportSpec{
		{
			Name: "_",
			Path: "a",
		},
		{Name: "al", Path: "b"},
		{Name: "", Path: "c"},
		{Name: ".", Path: "d"},
	}
	testCases := []struct {
		name       string
		importPath string
		want       string
	}{
		{"blank import has no qualifier", "a", ""},
		{"alias wins", "b", "al"},
		{"package clause name", "c", "pkg"},
		{"dot import has no qualifier", "d", ""},
		{"absent", "z", ""},
		{"empty path", "", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := LocalName(imports, testCase.importPath, "pkg"); got != testCase.want {
				t.Fatalf("localName() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func moduleNamed(path string) Module {
	return WalkModule(".", func(string) (string, error) { return "module " + path + "\n", nil })
}

func TestImportScope_Allowed(t *testing.T) {
	t.Parallel()

	mod := moduleNamed("m")
	scope := ImportScope{
		OldMod:   mod,
		NewMod:   mod,
		DropOK:   []string{"m/gone"},
		AddOK:    []string{"m/fresh"},
		DirMoves: map[string]string{"old": "new"},
	}
	testCases := []struct {
		name       string
		oldImports []ImportSpec
		newImports []ImportSpec
		want       bool
	}{
		{"unchanged", []ImportSpec{{"", "os"}}, []ImportSpec{{"", "os"}}, true},
		{
			"reordered",
			[]ImportSpec{{"", "os"}, {"", "io"}},
			[]ImportSpec{{"", "io"}, {"", "os"}},
			true,
		},
		{"unlisted drop", []ImportSpec{{"", "os"}}, nil, false},
		{"listed drop", []ImportSpec{{"", "m/gone"}}, nil, true},
		{"unlisted addition", nil, []ImportSpec{{"", "os"}}, false},
		{"listed addition", nil, []ImportSpec{{"", "m/fresh"}}, true},
		{
			"moved package under the same name",
			[]ImportSpec{{"x", "m/old"}},
			[]ImportSpec{{"x", "m/new"}},
			true,
		},
		{
			"moved package under another name",
			[]ImportSpec{{"x", "m/old"}},
			[]ImportSpec{{"y", "m/new"}},
			false,
		},
		{
			"rewrite to an unrelated dir",
			[]ImportSpec{{"", "m/old"}},
			[]ImportSpec{{"", "m/other"}},
			false,
		},
		{"locked blank import dropped", []ImportSpec{{"_", "m/gone"}}, nil, false},
		{"locked dot import added", nil, []ImportSpec{{".", "m/fresh"}}, false},
		{"locked cgo added", nil, []ImportSpec{{"", "C"}}, false},
		{
			"duplicate import counts twice",
			[]ImportSpec{{"", "os"}, {"", "os"}},
			[]ImportSpec{{"", "os"}},
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := scope.Allowed(
				testCase.oldImports,
				testCase.newImports,
			); got != testCase.want {
				t.Fatalf("Allowed() = %v, want %v", got, testCase.want)
			}
		})
	}
}
