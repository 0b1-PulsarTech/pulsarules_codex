package singleletter

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

type testCase struct {
	name   string
	source string
	want   []string
}

func defaultSettings() settings {
	return settings{reporter: NewAnalyzer().reporter()}
}

func assertMessages(t *testing.T, source string, cfg settings, want []string) {
	t.Helper()

	cache := astcache.New()
	file, err := cache.Parse("pkg.go", []byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fc := core.FileChange{Path: "pkg.go", Extension: ".go"}
	got := checkFile(cache.FileSet(), fc, file, cfg)
	if len(got) != len(want) {
		t.Fatalf("findings = %+v, want %q", got, want)
	}
	for pos, finding := range got {
		if finding.Message != want[pos] {
			t.Fatalf("message = %q, want %q", finding.Message, want[pos])
		}
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		change core.FileChange
		expect int
	}{
		{"regular file", core.FileChange{Path: "pkg.go", Extension: ".go"}, 1},
		{
			"generated file is skipped",
			core.FileChange{Path: "pkg.go", Extension: ".go", Generated: true},
			0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("pkg.go", []byte("package pkg\n\nvar x int\n")); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &core.AnalysisContext{
				ASTCache:     cache,
				ChangedFiles: []core.FileChange{testCase.change},
			}
			got := NewAnalyzer().Analyze(ctx)
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	if analyzer.ID() != "single-letter-name" || analyzer.Stage() != core.StageAST {
		t.Fatalf("id = %q stage = %v", analyzer.ID(), analyzer.Stage())
	}
	if got := analyzer.Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Fatalf("findings without an AST cache = %+v", got)
	}
}

func TestFindingHint(t *testing.T) {
	t.Parallel()

	cache := astcache.New()
	file, err := cache.Parse("pkg.go", []byte("package pkg\n\nvar x int\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fc := core.FileChange{Path: "pkg.go", Extension: ".go"}
	got := checkFile(cache.FileSet(), fc, file, defaultSettings())
	if len(got) != 1 || got[0].Line != 3 ||
		!strings.Contains(got[0].Suggestion, "what it is or holds; a short abbreviation") {
		t.Fatalf("findings = %+v", got)
	}
}

func TestAnalyzeExportedOnly(t *testing.T) {
	t.Parallel()

	const source = "package pkg\n\nconst X = 1\n\ntype T struct{}\n\nvar y = 2\n\n" +
		"func run(p int) { q := p; _ = q }\n"
	every := []string{
		`constant "X" is a single letter`,
		`type "T" is a single letter`,
		`variable "y" is a single letter`,
		`parameter "p" is a single letter`,
		`variable "q" is a single letter`,
	}
	testCases := []struct {
		name   string
		params map[string]any
		want   []string
	}{
		{"default flags every single letter", nil, every},
		{"explicit false flags every", map[string]any{"exported_only": false}, every},
		{"exported only keeps upper case", map[string]any{"exported_only": true}, []string{
			`constant "X" is a single letter`,
			`type "T" is a single letter`,
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("pkg.go", []byte(source)); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &core.AnalysisContext{
				ASTCache:     cache,
				ChangedFiles: []core.FileChange{{Path: "pkg.go", Extension: ".go"}},
				Config: &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
					"single-letter-name": {Enabled: true, Params: testCase.params},
				}},
			}
			var got []string
			for _, finding := range NewAnalyzer().Analyze(ctx) {
				got = append(got, finding.Message)
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("messages = %q, want %q", got, testCase.want)
			}
		})
	}
}
