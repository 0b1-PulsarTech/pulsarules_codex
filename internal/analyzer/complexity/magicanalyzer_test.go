package complexity

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestMagicNumberAnalyzer_Analyze(t *testing.T) {
	t.Parallel()

	testCases := append(checkFileCasesMagicNumbers(), checkFileTestCase{
		name:   "bodyless declaration has nothing to flag",
		source: "package foo\nfunc f() int\n",
		expect: 0,
	})
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("foo.go", []byte(testCase.source)); err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := NewMagicNumberAnalyzer().Analyze(&pass.Context{
				ASTCache:     cache,
				ChangedFiles: []pass.FileChange{{Path: "foo.go", Extension: ".go"}},
			})
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
			for _, hit := range got {
				if hit.AnalyzerID != MagicID || hit.Severity != finding.SeverityInfo {
					t.Fatalf("finding = %+v, want id %q at info", hit, MagicID)
				}
			}
		})
	}
}

func TestMagicNumberAnalyzer_NoCache(t *testing.T) {
	t.Parallel()

	if got := NewMagicNumberAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Fatalf("findings = %v, want none without an AST cache", got)
	}
}

func TestDescribeMagicNumber(t *testing.T) {
	t.Parallel()

	got := DescribeMagicNumber()
	if got.ID != "magic-number" || got.Rule != "code-smells" {
		t.Fatalf("descriptor = %+v", got)
	}
}

type numbersProvider struct{ numbers []syntax.Number }

func (numbersProvider) Languages() []language.ID { return []language.ID{language.Dart, language.Go} }

func (provider numbersProvider) Summarize(
	_ context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	files := map[string]*syntax.File{}
	for _, src := range corpus.Sources {
		files[src.Path] = &syntax.File{
			Path:     src.Path,
			Language: language.ForExtension(filepath.Ext(src.Path)).ID,
			Numbers:  provider.numbers,
		}
	}
	return files, nil
}

func numbersContext(change pass.FileChange, numbers ...syntax.Number) *pass.Context {
	return numbersContextFor([]pass.FileChange{change}, numbers...)
}

func numbersContextFor(changes []pass.FileChange, numbers ...syntax.Number) *pass.Context {
	sources := make([]syntax.Source, 0, len(changes))
	for _, change := range changes {
		sources = append(sources, syntax.Source{Path: change.Path})
	}
	read := func(string) ([]byte, bool) { return nil, true }
	return &pass.Context{
		ChangedFiles: changes,
		Syntax: syntax.NewCache(
			[]syntax.Provider{numbersProvider{numbers: numbers}},
			syntax.Corpus{Sources: sources, Read: read},
		),
	}
}

func allowedConfig(values ...any) *pass.AnalysisConfig {
	return &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
		MagicID: {Enabled: true, Params: map[string]any{paramAllowedNumbers: values}},
	}}
}

type syntaxCase struct {
	name    string
	path    string
	isTest  bool
	config  *pass.AnalysisConfig
	numbers []string
	want    []string
}

func (testCase syntaxCase) wantLines() []int {
	lines := make([]int, 0, len(testCase.want))
	for idx, value := range testCase.numbers {
		if len(lines) < len(testCase.want) && value == testCase.want[len(lines)] {
			lines = append(lines, idx+1)
		}
	}
	return lines
}

func syntaxCases() []syntaxCase {
	const logic = "pkg/lib/src/geometry/scale.dart"
	return []syntaxCase{
		{"literal in a logic file", logic, false, nil, []string{"0.5"}, []string{"0.5"}},
		{"two is allowed by default", logic, false, nil, []string{"2"}, nil},
		{
			"two flagged once the list is replaced", logic, false,
			allowedConfig("0", "1"),
			[]string{"2", "1"},
			[]string{"2"},
		},
		{"hex literal is flagged", logic, false, nil, []string{"0x10"}, []string{"0x10"}},
		{
			"hex spelling of an allowed value is flagged", logic, false,
			nil,
			[]string{"0x01", "1"},
			[]string{"0x01"},
		},
		{
			"float spelling must match the list", logic, false,
			nil,
			[]string{"1.0", "1.00"},
			[]string{"1.00"},
		},
		{
			"digit separator is a different spelling", logic, false,
			nil,
			[]string{"1000", "1_000"},
			[]string{"1_000"},
		},
		{
			"sign is never part of the literal, so -5 arrives as 5",
			logic,
			false,
			nil,
			[]string{"1", "5"},
			[]string{"5"},
		},
		{"test file is skipped", "pkg/test/scale_test.dart", true, nil, []string{"0.5"}, nil},
		{
			"Go syntax numbers are left to the AST walk", "pkg/scale.go", false,
			nil,
			[]string{"0.5"},
			nil,
		},
		{
			"empty configured list keeps the default",
			logic,
			false,
			allowedConfig(),
			[]string{"2"},
			nil,
		},
	}
}

func TestMagicNumberAnalyzer_SyntaxFiles(t *testing.T) {
	t.Parallel()

	for _, testCase := range syntaxCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			numbers := make([]syntax.Number, 0, len(testCase.numbers))
			for idx, value := range testCase.numbers {
				numbers = append(numbers, syntax.Number{Value: value, Line: idx + 1})
			}
			ctx := numbersContext(
				pass.FileChange{
					Path:      testCase.path,
					Extension: filepath.Ext(testCase.path),
					IsTest:    testCase.isTest,
				},
				numbers...)
			ctx.Config = testCase.config
			got := make([]string, 0, len(testCase.want))
			gotLines := make([]int, 0, len(testCase.want))
			for _, hit := range NewMagicNumberAnalyzer().Analyze(ctx) {
				if hit.AnalyzerID != MagicID || hit.Severity != finding.SeverityInfo ||
					hit.Suggestion != "name it with a const declaration" ||
					hit.File != testCase.path {
					t.Fatalf("finding = %+v", hit)
				}
				got = append(got, strings.TrimPrefix(hit.Message, "magic number "))
				gotLines = append(gotLines, hit.Line)
			}
			if strings.Join(got, ",") != strings.Join(testCase.want, ",") {
				t.Fatalf("flagged = %v, want %v", got, testCase.want)
			}
			if !slices.Equal(gotLines, testCase.wantLines()) {
				t.Fatalf("lines = %v, want %v", gotLines, testCase.wantLines())
			}
		})
	}
}

func TestMagicNumberAnalyzer_DisabledOverrideSilencesOneDirectory(t *testing.T) {
	t.Parallel()

	off := false
	rules := []pass.Override{
		pass.NewOverride([]string{"**/tokens/**"}, map[string]pass.OverrideConfig{
			MagicID: {Enabled: &off},
		}),
	}
	changes := []pass.FileChange{
		{Path: "pkg/lib/src/geometry/scale.dart", Extension: ".dart"},
		{Path: "pkg/lib/src/tokens/table.dart", Extension: ".dart"},
	}
	ctx := numbersContextFor(changes, syntax.Number{Value: "0.5", Line: 3})
	ctx.Config = &pass.AnalysisConfig{Overrides: rules}

	all := NewMagicNumberAnalyzer().Analyze(ctx)
	if len(all) != len(changes) {
		t.Fatalf("analyzer findings = %v, want one per file before overrides", all)
	}
	kept := overrides.Apply(all, MagicID, rules)
	if len(kept) != 1 || kept[0].File != "pkg/lib/src/geometry/scale.dart" {
		t.Fatalf("kept = %v, want only scale.dart", kept)
	}
}

func TestMagicNumberAnalyzer_GoAllowedNumbers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		source  string
		allowed []any
		want    int
	}{
		{"configured value is allowed", "x := 42", []any{"42"}, 0},
		{"list replaces the default", "x := 2", []any{"0", "1"}, 1},
		{"default holds without config", "x := 2", nil, 0},
		{"octal stays exempt", "x := 0o750", []any{"0"}, 0},
		{"strconv bit size stays exempt", "x, _ := strconv.ParseInt(\"1\", 0, 64)", []any{"0"}, 0},
		{"unary minus leaves the bare literal", "x := -5", []any{"-5"}, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := "package foo\nimport \"strconv\"\nvar _ = strconv.Itoa\n" +
				"func f() {\n\t" + testCase.source + "\n\t_ = x\n}\n"
			cache := astcache.New()
			if _, err := cache.Parse("foo.go", []byte(source)); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &pass.Context{
				ASTCache:     cache,
				ChangedFiles: []pass.FileChange{{Path: "foo.go", Extension: ".go"}},
			}
			if testCase.allowed != nil {
				ctx.Config = allowedConfig(testCase.allowed...)
			}
			if got := NewMagicNumberAnalyzer().Analyze(ctx); len(got) != testCase.want {
				t.Fatalf("findings = %v, want %d", got, testCase.want)
			}
		})
	}
}
