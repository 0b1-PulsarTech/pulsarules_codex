package registry

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

const fixtureDir = "testdata/paramscan"

type fixtureCase struct {
	name, dir, source string
	wanted            bool
	want              []string
	declared          map[string][]string
	with              []string
}

func listFixtures(fsys fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("list fixtures: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go.txt") {
			return nil, fmt.Errorf("%s/%s is not a *.go.txt fixture", dir, entry.Name())
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no fixtures under %s", dir)
	}
	return names, nil
}

func parseFixture(name, source string) (fixtureCase, error) {
	testCase := fixtureCase{
		name: name, source: source, declared: map[string][]string{"file-size": {"max_lines"}},
	}
	for line := range strings.Lines(source) {
		comment, isComment := strings.CutPrefix(line, "//")
		if !isComment {
			break
		}
		key, value, hasValue := strings.Cut(strings.TrimSpace(comment), ": ")
		if !hasValue {
			return fixtureCase{}, fmt.Errorf("%s: malformed header line %q", name, line)
		}
		if err := testCase.set(key, value); err != nil {
			return fixtureCase{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	switch {
	case testCase.dir == "":
		return fixtureCase{}, fmt.Errorf("%s: missing dir header", name)
	case !testCase.wanted:
		return fixtureCase{}, fmt.Errorf("%s: missing want header", name)
	}
	return testCase, nil
}

func (testCase *fixtureCase) set(key, value string) error {
	switch key {
	case "dir":
		testCase.dir = value
	case "want":
		return testCase.addWant(value)
	case "with":
		testCase.with = strings.Split(value, ",")
	case "declared":
		testCase.declared = map[string][]string{}
		for entry := range strings.SplitSeq(value, ";") {
			id, keys, _ := strings.Cut(entry, "=")
			testCase.declared[id] = strings.Split(keys, ",")
		}
	default:
		return fmt.Errorf("unknown header %q", key)
	}
	return nil
}

func (testCase *fixtureCase) addWant(value string) error {
	if value == "clean" {
		if len(testCase.want) > 0 {
			return errors.New("want: clean follows a problem")
		}
		testCase.wanted = true
		return nil
	}
	if testCase.wanted && len(testCase.want) == 0 {
		return errors.New("want: a problem follows clean")
	}
	testCase.wanted = true
	testCase.want = append(testCase.want, value)
	return nil
}

func (testCase fixtureCase) packages(t *testing.T) []*typedPackage {
	t.Helper()
	sources := map[string]string{testCase.dir: testCase.source}
	for _, name := range testCase.with {
		sources[name] = mustRead(t, "with", name+".go.txt")
	}
	fset, dirs := token.NewFileSet(), map[string][]*ast.File{}
	for dir, source := range sources {
		file, err := parser.ParseFile(fset, dir+".go", source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		dirs[dir] = append(dirs[dir], file)
	}
	return typeCheck(t, fset, dirs, fixtureImporter{})
}

func mustRead(t *testing.T, elems ...string) string {
	t.Helper()
	raw, err := fs.ReadFile(os.DirFS(fixtureDir), strings.Join(elems, "/"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(raw)
}

func TestScanParamsFixtures(t *testing.T) {
	t.Parallel()

	names, listErr := listFixtures(os.DirFS(fixtureDir), "cases")
	if listErr != nil {
		t.Fatal(listErr)
	}
	dirIDs := map[string][]string{"x": {"file-size"}, "y": {"text-markers", "typo"}}
	for _, name := range names {
		source := mustRead(t, "cases", name)
		testCase, err := parseFixture(strings.TrimSuffix(name, ".go.txt"), source)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := scanParams(testCase.packages(t), dirIDs)
			got := compareParams(result, testCase.declared)
			want := slices.Sorted(slices.Values(testCase.want))
			if !slices.Equal(got, want) {
				t.Fatalf("problems = %q, want exactly %q", got, want)
			}
		})
	}
}

func errContains(err error, text string) bool {
	return err != nil && strings.Contains(err.Error(), text)
}

func TestParseFixture(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		source  string
		wantErr string
	}{
		{"a problem", "// dir: x\n// want: p\npackage x\n", ""},
		{"a clean case", "// dir: x\n// want: clean\npackage x\n", ""},
		{"two problems", "// dir: x\n// want: p\n// want: q\npackage x\n", ""},
		{"a misspelt header", "// dir: x\n// wnat: p\npackage x\n", `unknown header "wnat"`},
		{"no dir", "// want: clean\npackage x\n", "missing dir header"},
		{"no want", "// dir: x\npackage x\n", "missing want header"},
		{"a misspelt want", "// dir: x\n// wnat: clean\npackage x\n", `unknown header "wnat"`},
		{"clean after a problem", "// dir: x\n// want: p\n// want: clean\n", "clean follows"},
		{"a problem after clean", "// dir: x\n// want: clean\n// want: p\n", "follows clean"},
		{
			"a comment without a colon in the header block",
			"// dir: x\n// stray note\n// want: clean\npackage x\n",
			"malformed header line",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseFixture("case", testCase.source)
			switch {
			case testCase.wantErr == "" && err != nil:
				t.Fatalf("parseFixture() error = %v, want nil", err)
			case testCase.wantErr != "" && !errContains(err, testCase.wantErr):
				t.Fatalf("parseFixture() error = %v, want %q", err, testCase.wantErr)
			}
		})
	}
}

func TestListFixtures(t *testing.T) {
	t.Parallel()

	file := &fstest.MapFile{Data: []byte("x")}
	testCases := []struct {
		name    string
		files   fstest.MapFS
		wantErr bool
	}{
		{"fixtures only", fstest.MapFS{"c/a.go.txt": file, "c/b.go.txt": file}, false},
		{"a stray go file", fstest.MapFS{"c/a.go.txt": file, "c/b.go": file}, true},
		{"a stray notes file", fstest.MapFS{"c/a.go.txt": file, "c/n.md": file}, true},
		{"a nested directory", fstest.MapFS{"c/a.go.txt": file, "c/d/b.go.txt": file}, true},
		{"nothing", fstest.MapFS{"c/.keep": file}, true},
		{"an empty directory", fstest.MapFS{"c": &fstest.MapFile{Mode: fs.ModeDir}}, true},
		{"no directory", fstest.MapFS{}, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := listFixtures(testCase.files, "c")
			if (err != nil) != testCase.wantErr {
				t.Fatalf("listFixtures() error = %v, want error %v", err, testCase.wantErr)
			}
		})
	}
}
