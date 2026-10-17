package gosrc

import "testing"

func TestWalkModule_Path(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		goMod string
		want  string
	}{
		{"plain", "module example.com/m\n\ngo 1.25\n", "example.com/m"},
		{"quoted", "module \"example.com/q\"\n", "example.com/q"},
		{"missing directive", "go 1.25\n", ""},
		{"empty", "", ""},
		{"syntax error elsewhere", "module example.com/x\n\nrequire (\n", "example.com/x"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			read := func(string) (string, error) { return testCase.goMod, nil }
			if got := WalkModule(".", read).ImportPath("."); got != testCase.want {
				t.Fatalf("WalkModule().ImportPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestModule_ImportPathAndDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		mod      Module
		dir      string
		wantPath string
		wantDir  string
	}{
		{"root module nested dir", Module{"m", "."}, "a/b", "m/a/b", "a/b"},
		{"root module root dir", Module{"m", "."}, ".", "m", "."},
		{"nested module", Module{"m/sub", "sub"}, "sub/x", "m/sub/x", "sub/x"},
		{"nested module root", Module{"m/sub", "sub"}, "sub", "m/sub", "sub"},
		{"no module", Module{}, "a", "", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			importPath := testCase.mod.ImportPath(testCase.dir)
			if importPath != testCase.wantPath {
				t.Fatalf("importPath() = %q, want %q", importPath, testCase.wantPath)
			}
			dir, ok := testCase.mod.DirOf(importPath)
			if ok != (testCase.wantDir != "") || dir != testCase.wantDir {
				t.Fatalf("dirOf(%q) = %q, %v, want %q", importPath, dir, ok, testCase.wantDir)
			}
		})
	}
}

func TestModule_DirOfRejectsForeignPaths(t *testing.T) {
	t.Parallel()

	mod := Module{"example.com/m", "."}
	for _, foreign := range []string{"example.com/mx/a", "github.com/evil/m", "fmt"} {
		if dir, ok := mod.DirOf(foreign); ok {
			t.Fatalf("dirOf(%q) = %q, want it rejected", foreign, dir)
		}
	}
}
