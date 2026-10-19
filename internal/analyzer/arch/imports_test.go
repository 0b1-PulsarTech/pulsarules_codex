package arch

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type discoverCase struct {
	name  string
	files map[string]string
	want  map[string][]string
}

func discoverCases(module string) []discoverCase {
	return []discoverCase{
		{
			name: "two files in one package union and dedupe imports",
			files: map[string]string{
				"pkg/a.go": goFile("pkg", module+"/x", module+"/y"),
				"pkg/b.go": goFile("pkg", module+"/x", module+"/z"),
			},
			want: map[string][]string{
				module + "/pkg": {module + "/x", module + "/y", module + "/z"},
			},
		},
		{
			name: "dot directory is skipped",
			files: map[string]string{
				"pkg/a.go":         goFile("pkg", module+"/x"),
				".hidden/h.go":     goFile("hidden", module+"/y"),
				"pkg/.nested/n.go": goFile("nested", module+"/w"),
			},
			want: map[string][]string{module + "/pkg": {module + "/x"}},
		},
		{
			name: "underscore-prefixed directory is skipped",
			files: map[string]string{
				"pkg/a.go":      goFile("pkg", module+"/x"),
				"_scratch/s.go": goFile("scratch", module+"/y"),
				"pkg/_gen/g.go": goFile("gen", module+"/w"),
			},
			want: map[string][]string{module + "/pkg": {module + "/x"}},
		},
		{
			name: "testdata is skipped",
			files: map[string]string{
				"pkg/a.go":          goFile("pkg", module+"/x"),
				"testdata/t.go":     goFile("t", module+"/y"),
				"pkg/testdata/p.go": goFile("p", module+"/w"),
			},
			want: map[string][]string{module + "/pkg": {module + "/x"}},
		},
		{
			name: "test files are skipped",
			files: map[string]string{
				"pkg/a.go":      goFile("pkg", module+"/x"),
				"pkg/a_test.go": goFile("pkg", module+"/t"),
			},
			want: map[string][]string{module + "/pkg": {module + "/x"}},
		},
		{
			name:  "root-level file maps to the module path",
			files: map[string]string{"main.go": goFile("main", module+"/x")},
			want:  map[string][]string{module: {module + "/x"}},
		},
	}
}

func headerDiscoverCases(module string) []discoverCase {
	return []discoverCase{
		{
			name: "malformed file does not halt discovery",
			files: map[string]string{
				"bad/b.go": "not go source",
				"pkg/a.go": goFile("pkg", module+"/x"),
			},
			want: map[string][]string{module + "/bad": nil, module + "/pkg": {module + "/x"}},
		},
		{
			name: "a syntax error after the import block keeps the file's imports",
			files: map[string]string{
				"pkg/a.go": "package pkg\nimport \"example.com/mod/x\"\nfunc broken( {\n",
			},
			want: map[string][]string{module + "/pkg": {module + "/x"}},
		},
		{
			name: "a stray token right after the import block keeps the imports",
			files: map[string]string{
				"pkg/a.go": "package pkg\nimport \"example.com/mod/x\"\n#\n",
			},
			want: map[string][]string{module + "/pkg": {module + "/x"}},
		},
		{
			name: "an import after a declaration is not an import",
			files: map[string]string{
				"pkg/a.go": "package pkg\nfunc f() {}\nimport \"example.com/mod/x\"\n",
				"pkg/b.go": "package pkg\nvar v = 1\nimport \"example.com/mod/y\"\n",
			},
			want: map[string][]string{module + "/pkg": nil},
		},
		{
			name: "a syntax error inside the import block drops the file's imports",
			files: map[string]string{
				"domain/broken.go": "package domain\n\nimport (\n\t\"example.com/mod/infra\"\n\t!!!\n)\n",
				"domain/ok.go":     goFile("domain", module+"/x"),
			},
			want: map[string][]string{module + "/domain": {module + "/x"}},
		},
		{
			name: "an unterminated import block drops the file's imports",
			files: map[string]string{
				"pkg/a.go": "package pkg\nimport (\n\t\"example.com/mod/x\"\n",
			},
			want: map[string][]string{module + "/pkg": nil},
		},
	}
}

func goFile(pkg string, imports ...string) string {
	var builder strings.Builder
	builder.WriteString("package " + pkg + "\n")
	for _, imp := range imports {
		builder.WriteString("import \"" + imp + "\"\n")
	}
	return builder.String()
}

func TestDiscoverPackages(t *testing.T) {
	t.Parallel()

	const module = "example.com/mod"
	testCases := slices.Concat(discoverCases(module), headerDiscoverCases(module))
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			for name, content := range testCase.files {
				writeFile(t, projectDir, name, content)
			}

			provider := source.NewProvider(projectDir, source.NewSkipSet(nil))
			pkgs := map[string][]string(
				loadImportGraph(gomod.NewIndex(provider), module).deps,
			)

			if !reflect.DeepEqual(pkgs, testCase.want) {
				t.Fatalf("got %v, want %v", pkgs, testCase.want)
			}
		})
	}
}

func TestLoadImportGraph_Sites(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "a/a1.go", "package a\n")
	writeFile(t, projectDir, "a/a2.go",
		"package a\n\nimport (\n\t\"fmt\"\n\t\"example.com/mod/b\"\n)\n")
	writeFile(t, projectDir, "a/a3.go", "package a\n\nimport \"example.com/mod/b\"\n")
	writeFile(t, projectDir, "b/b.go", "package b\n")

	graph := loadImportGraph(gomod.NewIndex(newSources(projectDir)), "example.com/mod")

	got := graph.sites[edge{from: "example.com/mod/a", to: "example.com/mod/b"}]
	want := importSite{file: "a/a2.go", line: 5}
	if got != want {
		t.Errorf("site = %+v, want the first import spec %+v", got, want)
	}
}

func TestFindCycles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		graph depGraph
		want  [][]string
	}{
		{
			name:  "no cycles",
			graph: depGraph{"a": {"b"}, "b": {"c"}, "c": nil},
			want:  nil,
		},
		{
			name:  "direct cycle",
			graph: depGraph{"a": {"b"}, "b": {"a"}},
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "self loop",
			graph: depGraph{"a": {"a"}},
			want:  [][]string{{"a"}},
		},
		{
			name:  "indirect cycle",
			graph: depGraph{"a": {"b"}, "b": {"c"}, "c": {"a"}},
			want:  [][]string{{"a", "b", "c"}},
		},
		{
			name:  "disjoint cycles come out in root order",
			graph: depGraph{"x": {"y"}, "y": {"x"}, "a": {"b"}, "b": {"a"}},
			want:  [][]string{{"a", "b"}, {"x", "y"}},
		},
		{
			name:  "dependencies are walked in sorted order",
			graph: depGraph{"a": {"c", "b"}, "b": {"c"}, "c": {"a"}},
			want:  [][]string{{"a", "b", "c"}},
		},
		{
			name:  "a cycle reached through a tail is rotated to its smallest member",
			graph: depGraph{"a": {"d"}, "d": {"b"}, "b": {"d"}},
			want:  [][]string{{"b", "d"}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := findCycles(testCase.graph); !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("findCycles = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestFindCycles_IsDeterministic(t *testing.T) {
	t.Parallel()

	graph := depGraph{
		"c": {"a"},
		"b": {"c"},
		"a": {"b"},
		"d": {"c", "b"},
	}
	want := [][]string{{"a", "b", "c"}}

	for range 50 {
		if got := findCycles(graph); !reflect.DeepEqual(got, want) {
			t.Fatalf("findCycles = %v, want %v", got, want)
		}
	}
}

func TestClassifyLayer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		pkg  string
		want string
	}{
		{"example.com/domain/foo", "domain"},
		{"example.com/infra/db", "infra"},
		{"example.com/transport/rest", "transport"},
		{"example.com/rest/v1", "transport"},
		{"example.com/grpc/v1", "transport"},
		{"example.com/cmd/server", "cmd"},
		{"example.com/domain/infra", "domain"},
		{"example.com/internal/util", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.pkg, func(t *testing.T) {
			t.Parallel()
			got, _ := classifyLayer(testCase.pkg)
			if got.name != testCase.want {
				t.Fatalf("classifyLayer(%q) = %q, want %q", testCase.pkg, got.name, testCase.want)
			}
		})
	}
}

func TestCheckBoundaries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		graph depGraph
		want  int
	}{
		{
			name: "no violations",
			graph: depGraph{
				"example.com/cmd/app":         {"example.com/internal/domain"},
				"example.com/infra/db":        {"example.com/internal/domain"},
				"example.com/internal/domain": nil,
			},
			want: 0,
		},
		{
			name: "domain imports infra",
			graph: depGraph{
				"example.com/internal/domain": {"example.com/infra/db"},
			},
			want: 1,
		},
		{
			name: "domain imports transport",
			graph: depGraph{
				"example.com/internal/domain": {"example.com/transport/rest"},
			},
			want: 1,
		},
		{
			name: "unclassified importer (apps/+libs/+proto/ " +
				"layout) reports nothing, not an inward violation",
			graph: depGraph{
				"example.com/apps/api": {"example.com/domain/thing"},
			},
			want: 0,
		},
		{
			name: "unclassified dependency reports nothing",
			graph: depGraph{
				"example.com/internal/domain": {"example.com/libs/store"},
			},
			want: 0,
		},
		{
			name: "an entirely unrecognized layout reports nothing",
			graph: depGraph{
				"example.com/apps/api":    {"example.com/libs/store", "example.com/proto/apiv1"},
				"example.com/libs/store":  {"example.com/proto/apiv1"},
				"example.com/proto/apiv1": nil,
			},
			want: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := checkBoundaries(importGraph{deps: testCase.graph}, "example.com")
			if len(got) != testCase.want {
				t.Fatalf("got %d violations, want %d: %v", len(got), testCase.want, got)
			}
		})
	}
}

func TestCheckBoundaries_IsDeterministic(t *testing.T) {
	t.Parallel()

	graph := depGraph{
		"example.com/domain/c": {"example.com/infra/db"},
		"example.com/domain/a": {"example.com/transport/rest"},
		"example.com/domain/b": {"example.com/infra/db", "example.com/cmd/app"},
	}
	want := []string{
		"domain/a (domain) imports transport/rest (transport): inner layer depends on outer",
		"domain/b (domain) imports infra/db (infra): inner layer depends on outer",
		"domain/b (domain) imports cmd/app (cmd): inner layer depends on outer",
		"domain/c (domain) imports infra/db (infra): inner layer depends on outer",
	}

	for range 50 {
		found := checkBoundaries(importGraph{deps: graph}, "example.com")
		got := make([]string, 0, len(found))
		for _, violation := range found {
			got = append(got, violation.message)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("checkBoundaries = %v, want %v", got, want)
		}
	}
}

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

//nolint:ireturn // test helper mirrors the provider factory
func newSources(dir string) source.Provider {
	return source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs()))
}

func TestLoadImportGraph(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "foo.go", "package foo\n")
	writeFile(t, projectDir, "bar/baz.go", "package bar\nimport \"example.com/mod/foo\"\n")

	got := loadImportGraph(gomod.NewIndex(newSources(projectDir)), "example.com/mod").deps
	want := depGraph{"example.com/mod": nil, "example.com/mod/bar": {"example.com/mod/foo"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
