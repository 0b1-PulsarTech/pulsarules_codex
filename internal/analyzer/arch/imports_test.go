package arch

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
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
				"pkg/a.go": goFile("pkg", "fmt", "os"),
				"pkg/b.go": goFile("pkg", "fmt", "strings"),
			},
			want: map[string][]string{module + "/pkg": {"fmt", "os", "strings"}},
		},
		{
			name: "dot directory is skipped",
			files: map[string]string{
				"pkg/a.go":         goFile("pkg", "fmt"),
				".hidden/h.go":     goFile("hidden", "os"),
				"pkg/.nested/n.go": goFile("nested", "io"),
			},
			want: map[string][]string{module + "/pkg": {"fmt"}},
		},
		{
			name: "underscore-prefixed directory is skipped",
			files: map[string]string{
				"pkg/a.go":      goFile("pkg", "fmt"),
				"_scratch/s.go": goFile("scratch", "os"),
				"pkg/_gen/g.go": goFile("gen", "io"),
			},
			want: map[string][]string{module + "/pkg": {"fmt"}},
		},
		{
			name: "vendor and node_modules are skipped",
			files: map[string]string{
				"pkg/a.go":              goFile("pkg", "fmt"),
				"vendor/v.go":           goFile("v", "os"),
				"web/node_modules/n.go": goFile("n", "io"),
			},
			want: map[string][]string{module + "/pkg": {"fmt"}},
		},
		{
			name: "test files are skipped",
			files: map[string]string{
				"pkg/a.go":      goFile("pkg", "fmt"),
				"pkg/a_test.go": goFile("pkg", "testing"),
			},
			want: map[string][]string{module + "/pkg": {"fmt"}},
		},
		{
			name:  "root-level file maps to the module path",
			files: map[string]string{"main.go": goFile("main", "fmt")},
			want:  map[string][]string{module: {"fmt"}},
		},
		{
			name: "malformed file does not halt discovery",
			files: map[string]string{
				"bad/b.go": "not go source",
				"pkg/a.go": goFile("pkg", "fmt"),
			},
			want: map[string][]string{module + "/pkg": {"fmt"}},
		},
		{
			name: "a syntax error after the import block keeps the file's imports",
			files: map[string]string{
				"pkg/a.go": "package pkg\nimport \"fmt\"\nfunc broken( {\n",
			},
			want: map[string][]string{module + "/pkg": {"fmt"}},
		},
	}
}

func goFile(pkg string, imports ...string) string {
	var b strings.Builder
	b.WriteString("package " + pkg + "\n")
	for _, imp := range imports {
		b.WriteString("import \"" + imp + "\"\n")
	}
	return b.String()
}

func TestDiscoverPackages(t *testing.T) {
	t.Parallel()

	const module = "example.com/mod"
	for _, testCase := range discoverCases(module) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			for name, content := range testCase.files {
				writeFile(t, projectDir, name, content)
			}

			provider := sources.NewSourceProvider(projectDir, sources.NewSkipSet(nil))
			pkgs := discoverPackages(provider, module)

			got := make(map[string][]string, len(pkgs))
			for key, info := range pkgs {
				got[key] = info.Imports
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestBuildGraph(t *testing.T) {
	t.Parallel()

	pkgs := map[string]*pkgImports{
		"example.com/a": {Imports: []string{"fmt", "example.com/b"}},
		"example.com/b": {Imports: []string{"example.com/c"}},
		"example.com/c": {Imports: []string{"os"}},
	}
	g := buildGraph(pkgs, "example.com")
	if len(g) != 3 {
		t.Fatalf("expected 3 entries in graph, got %d", len(g))
	}
	if !slices.Contains(g["example.com/a"], "example.com/b") {
		t.Error("example.com/a should depend on example.com/b")
	}
}

func TestFindCycles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		graph depGraph
		wantN int
	}{
		{
			name: "no cycles",
			graph: depGraph{
				"a": {"b"},
				"b": {"c"},
				"c": nil,
			},
			wantN: 0,
		},
		{
			name: "direct cycle",
			graph: depGraph{
				"a": {"b"},
				"b": {"a"},
			},
			wantN: 1,
		},
		{
			name: "self loop",
			graph: depGraph{
				"a": {"a"},
			},
			wantN: 1,
		},
		{
			name: "indirect cycle",
			graph: depGraph{
				"a": {"b"},
				"b": {"c"},
				"c": {"a"},
			},
			wantN: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cycles := findCycles(testCase.graph)
			if len(cycles) != testCase.wantN {
				t.Fatalf("got %d cycles, want %d: %v", len(cycles), testCase.wantN, cycles)
			}
		})
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
		{"example.com/internal/util", layerUnclassified},
	}
	for _, testCase := range testCases {
		t.Run(testCase.pkg, func(t *testing.T) {
			t.Parallel()
			got := classifyLayer(testCase.pkg)
			if got != testCase.want {
				t.Fatalf("classifyLayer(%q) = %q, want %q", testCase.pkg, got, testCase.want)
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
			got := checkBoundaries(testCase.graph, "example.com")
			if len(got) != testCase.want {
				t.Fatalf("got %d violations, want %d: %v", len(got), testCase.want, got)
			}
		})
	}
}

func TestDiscoverRealProject(t *testing.T) {
	t.Parallel()
	t.Skip("runs against the actual project tree - enable manually")
	pkgs := discoverPackages(newSources("."), "github.com/0b1-PulsarTech/pulsarules_codex")
	if len(pkgs) == 0 {
		t.Fatal("no packages discovered")
	}
	g := buildGraph(pkgs, "github.com/0b1-PulsarTech/pulsarules_codex")
	cycles := findCycles(g)
	if len(cycles) > 0 {
		for _, c := range cycles {
			t.Errorf("cycle: %s", strings.Join(c, " → "))
		}
	}
	violations := checkBoundaries(g, "github.com/0b1-PulsarTech/pulsarules_codex")
	if len(violations) > 0 {
		for _, v := range violations {
			t.Logf("boundary: %s", v)
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
func newSources(dir string) sources.SourceProvider {
	return sources.NewSourceProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs()))
}
