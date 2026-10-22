package rulesummary

import (
	"maps"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func testRules() map[string]string {
	return registry.Rules()
}

func loadIndex(t *testing.T) *knowledge.Index {
	t.Helper()
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return idx
}

func sortedKeys[Value any](source map[string]Value) []string {
	return slices.Sorted(maps.Keys(source))
}

func TestMapTargetsExist(t *testing.T) {
	t.Parallel()

	type targetCase struct{ name, ruleID string }
	testCases := []targetCase{{"golangci-lint prefix", "code-smells"}}
	for _, analyzerID := range sortedKeys(testRules()) {
		ruleID := testRules()[analyzerID]
		testCases = append(testCases, targetCase{"default " + analyzerID, ruleID})
	}
	for _, analyzerID := range sortedKeys(languageRules()) {
		for lang, ruleID := range languageRules()[analyzerID] {
			name := "language " + analyzerID + " " + string(lang)
			testCases = append(testCases, targetCase{name, ruleID})
		}
	}
	for _, analyzerID := range sortedKeys(neutralFallbacks()) {
		ruleID := neutralFallbacks()[analyzerID]
		testCases = append(testCases, targetCase{"fallback " + analyzerID, ruleID})
	}

	idx := loadIndex(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if !idx.RuleExists(testCase.ruleID) {
				t.Fatalf("rule %q is not in the knowledge index", testCase.ruleID)
			}
		})
	}
}

func TestFallbacksAreNeutral(t *testing.T) {
	t.Parallel()

	type fallbackCase struct{ analyzerID, ruleID string }
	fallbackIDs := sortedKeys(neutralFallbacks())
	testCases := make([]fallbackCase, 0, len(fallbackIDs))
	for _, analyzerID := range fallbackIDs {
		testCases = append(testCases, fallbackCase{analyzerID, neutralFallbacks()[analyzerID]})
	}

	idx := loadIndex(t)
	known := languageTags()
	for _, testCase := range testCases {
		t.Run(testCase.analyzerID, func(t *testing.T) {
			t.Parallel()
			rule, _ := idx.Rule(testCase.ruleID)
			for _, tag := range rule.Tags {
				if known[tag] {
					t.Fatalf("fallback %q is tagged %q", testCase.ruleID, tag)
				}
			}
		})
	}
}

func TestLanguageTags(t *testing.T) {
	t.Parallel()

	want := []string{"dart", "go", "javascript", "svelte", "typescript"}
	if got := sortedKeys(languageTags()); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

type compatCase struct{ analyzerID, file, language string }

func compatCases() []compatCase {
	testCases := []struct{ file, language string }{
		{"a.go", "go"},
		{"a.dart", "dart"},
		{"a.ts", "typescript"},
		{"a.tsx", "typescript"},
		{"a.mjs", "javascript"},
		{"a.cjs", "javascript"},
		{"a.jsx", "javascript"},
		{"a.js", "javascript"},
		{"a.svelte", "svelte"},
		{"a.vue", ""},
		{"a.md", ""},
		{"a.sql", ""},
		{"a.proto", ""},
		{"a.yaml", ""},
		{"a.xyz", ""},
	}
	var cases []compatCase
	analyzerIDs := append(sortedKeys(testRules()), "golangci-lint/gocyclo")
	for _, analyzerID := range analyzerIDs {
		if goPinned(analyzerID) {
			continue
		}
		for _, testCase := range testCases {
			cases = append(cases, compatCase{analyzerID, testCase.file, testCase.language})
		}
	}
	return cases
}

func TestGoPinnedSet(t *testing.T) {
	t.Parallel()

	pinned := map[string]bool{"import-cycle": true, "golangci-lint": true}
	analyzerIDs := append(sortedKeys(testRules()), "golangci-lint/gocyclo")
	type pinCase struct {
		analyzerID string
		want       bool
	}
	testCases := make([]pinCase, 0, len(analyzerIDs))
	for _, analyzerID := range analyzerIDs {
		want := pinned[analyzerID] || analyzerID == "golangci-lint/gocyclo"
		testCases = append(testCases, pinCase{analyzerID, want})
	}
	for _, testCase := range testCases {
		t.Run(testCase.analyzerID, func(t *testing.T) {
			t.Parallel()
			if got := goPinned(testCase.analyzerID); got != testCase.want {
				t.Fatalf("goPinned = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestResolveIsTagCompatible(t *testing.T) {
	t.Parallel()

	testCases := compatCases()

	idx := loadIndex(t)
	known := languageTags()
	resolver := newResolver(idx, testRules())
	for _, testCase := range testCases {
		t.Run(testCase.analyzerID+" "+testCase.file, func(t *testing.T) {
			t.Parallel()
			ruleID := resolver.resolve(testCase.analyzerID, testCase.file)
			if ruleID == "" {
				return
			}
			rule, _ := idx.Rule(ruleID)
			for _, tag := range rule.Tags {
				if known[tag] && tag != testCase.language {
					t.Fatalf("%s on %s: %q tagged %q", testCase.analyzerID, testCase.file,
						ruleID, tag)
				}
			}
		})
	}
}

type resolveCase struct {
	name       string
	analyzerID string
	file       string
	want       string
}

func resolveCases() []resolveCase {
	return []resolveCase{
		{"go file size", "file-size", "a.go", "source-files"},
		{"dart file size", "file-size", "lib/a.dart", "source-files"},
		{"svelte text markers", "text-markers", "src/a.svelte", "source-files"},
		{"mjs typographic", "typographic-markers", "a.mjs", "source-files"},
		{"markdown file size", "file-size", "README.md", "source-files"},
		{"go doc comment", "doc-comment", "a.go", "effective-go"},
		{"dart doc comment", "doc-comment", "a.dart", "comments"},
		{"ts doc comment", "doc-comment", "a.ts", "comments"},
		{"go import groups", "import-groups", "a.go", "imports"},
		{"dart import groups", "import-groups", "a.dart", "dart-style"},
		{"ts import groups", "import-groups", "a.ts", ""},
		{"go package files", "package-file-count", "a.go", "code-placement-go"},
		{"dart package files", "package-file-count", "a.dart", "code-placement"},
		{"ts package files", "package-file-count", "a.ts", "code-placement"},
		{"dart complexity", "complexity", "a.dart", ""},
		{"go complexity", "complexity", "a.go", "code-smells"},
		{"dart single letter", "single-letter-name", "a.dart", ""},
		{"dart top of file", "top-of-file", "a.dart", "comments"},
		{"markdown naming", "naming", "a.md", ""},
		{"sql control flow", "control-flow", "a.sql", ""},
		{"unknown extension naming", "naming", "a.xyz", ""},
		{"unknown extension size", "file-size", "a.xyz", "source-files"},
		{"no path commit", "commit-lint", "", "commits"},
		{"no path linter", "golangci-lint/gocyclo", "", "code-smells"},
		{"no path complexity", "complexity", "", "code-smells"},
		{"unknown analyzer", "no-such-analyzer", "a.dart", ""},
		{"pinned linter on dart", "golangci-lint/gocyclo", "a.dart", "code-smells"},
		{"import cycle on dot", "import-cycle", ".", "module-boundaries"},
		{"linter config", "golangci-lint/gocyclo", ".golangci.yml", "code-smells"},
		{"linter on go.mod", "golangci-lint/typecheck", "go.mod", "code-smells"},
		{"linter failure on go.mod", "golangci-lint", "go.mod", "code-smells"},
		{"import cycle on go.mod", "import-cycle", "go.mod", "module-boundaries"},
		{"import cycle on dart", "import-cycle", "a.dart", "module-boundaries"},
		{"cjs file size", "file-size", "a.cjs", "source-files"},
		{"jsx doc comment", "doc-comment", "a.jsx", "comments"},
		{"vue package files", "package-file-count", "a.vue", "code-placement"},
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	resolver := newResolver(loadIndex(t), testRules())
	for _, testCase := range resolveCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := resolver.resolve(testCase.analyzerID, testCase.file)
			if got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAttachKeepsPresetRule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		analyzerID  string
		file        string
		presetRule  string
		wantRule    string
		wantSummary bool
	}{
		{"go-tagged preset on dart", "file-size", "a.dart", "code-smells", "code-smells", true},
		{"delegated on svelte", "file-size", "a.svelte", "naming", "naming", true},
		{"preset on no path", "file-size", "", "imports", "imports", true},
		{"no preset on dart", "file-size", "a.dart", "", "source-files", true},
		{"no preset unmappable", "complexity", "a.dart", "", "", false},
	}

	idx := loadIndex(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			findings := []finding.Finding{{
				AnalyzerID: testCase.analyzerID, File: testCase.file, RuleID: testCase.presetRule,
			}}
			Attach(idx, testRules())(findings)
			if findings[0].RuleID != testCase.wantRule {
				t.Fatalf("RuleID %q, want %q", findings[0].RuleID, testCase.wantRule)
			}
			if (findings[0].RuleSummary != "") != testCase.wantSummary {
				t.Fatalf("summary %q, want present=%v", findings[0].RuleSummary,
					testCase.wantSummary)
			}
		})
	}
}

func TestFitsRejectsLanguageMismatch(t *testing.T) {
	t.Parallel()

	resolver := newResolver(loadIndex(t), testRules())
	testCases := []struct {
		name     string
		ruleID   string
		language language.ID
		want     bool
	}{
		{"go rule on go", "naming", language.Go, true},
		{"go rule on dart", "naming", language.Dart, false},
		{"go rule on no language", "naming", "", false},
		{"dart rule on dart", "dart-style", language.Dart, true},
		{"dart rule on go", "dart-style", language.Go, false},
		{"svelte rule on svelte", "atomic-design", language.Svelte, true},
		{"svelte rule on ts", "atomic-design", language.TypeScript, false},
		{"neutral on dart", "comments", language.Dart, true},
		{"neutral on no language", "source-files", "", true},
		{"unknown rule", "no-such-rule", language.Dart, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := resolver.fits(testCase.ruleID, testCase.language); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}
