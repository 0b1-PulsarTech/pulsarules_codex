package arch

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDiscoverPackages(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "foo.go", "package foo\n")
	writeFile(t, projectDir, "bar/baz.go", "package bar\n")

	pkgs := discoverPackages(projectDir, "example.com/mod")
	if len(pkgs) == 0 {
		t.Fatal("expected at least one package")
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
			// The regression: an apps/ + libs/ + proto/ tree matches no layer
			// rule, and ranking it innermost turned every ordinary import into
			// a reported inward violation.
			name: "unclassified importer reports nothing",
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
	// simplification: real-project discovery skips walking past test boundaries; it
	// relies on the unit tests above. Upgrade path: use go/packages for
	// accurate multi-module import resolution.
	t.Skip("runs against the actual project tree - enable manually")
	pkgs := discoverPackages(".", "github.com/0b1-PulsarTech/pulsarules_codex")
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

// helpers

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
