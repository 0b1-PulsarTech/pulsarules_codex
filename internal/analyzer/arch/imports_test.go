package arch

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

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
	testCases := discoverCases(module)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			for name, content := range testCase.files {
				writeFile(t, projectDir, name, content)
			}

			provider := source.NewProvider(projectDir, source.NewSkipSet(nil))
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
	graph := buildGraph(pkgs, "example.com")
	if len(graph) != 3 {
		t.Fatalf("expected 3 entries in graph, got %d", len(graph))
	}
	if !slices.Contains(graph["example.com/a"], "example.com/b") {
		t.Error("example.com/a should depend on example.com/b")
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
		if got := checkBoundaries(graph, "example.com"); !reflect.DeepEqual(got, want) {
			t.Fatalf("checkBoundaries = %v, want %v", got, want)
		}
	}
}

func TestDiscoverRealProject(t *testing.T) {
	t.Parallel()
	t.Skip("runs against the actual project tree - enable manually")
	pkgs := discoverPackages(newSources("."), "github.com/0b1-PulsarTech/pulsarules_codex")
	if len(pkgs) == 0 {
		t.Fatal("no packages discovered")
	}
	graph := buildGraph(pkgs, "github.com/0b1-PulsarTech/pulsarules_codex")
	cycles := findCycles(graph)
	if len(cycles) > 0 {
		for _, cycle := range cycles {
			t.Errorf("cycle: %s", strings.Join(cycle, " → "))
		}
	}
	violations := checkBoundaries(graph, "github.com/0b1-PulsarTech/pulsarules_codex")
	if len(violations) > 0 {
		for _, violation := range violations {
			t.Logf("boundary: %s", violation)
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
