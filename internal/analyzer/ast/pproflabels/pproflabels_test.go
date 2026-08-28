package pproflabels

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *core.AnalysisContext
		expect int
	}{
		{
			name: "no cache and no files",
			ctx:  &core.AnalysisContext{},
		},
		{
			name: "non-go file skipped",
			ctx: &core.AnalysisContext{
				ASTCache:     astcache.New(),
				ChangedFiles: []core.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := a.Analyze(testCase.ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

type checkFileTestCase struct {
	name       string
	source     string
	expect     int
	wantSubstr string
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(piiKeyFormCases(), checkFileCleanCases()...)
	testCases = append(testCases, pprofImportScopeCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyzeSource(t, testCase.source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
			if testCase.wantSubstr != "" &&
				!strings.Contains(got[0].Message, testCase.wantSubstr) {
				t.Fatalf("message = %q, want substring %q", got[0].Message, testCase.wantSubstr)
			}
		})
	}
}

func piiKeyFormCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:       "an exact PII key is flagged",
			source:     labelSource(`"email", email`),
			expect:     1,
			wantSubstr: `"email"`,
		},
		{
			name:   "an underscored PII key is flagged",
			source: labelSource(`"user_id", userID`),
			expect: 1,
		},
		{
			name:   "a hyphenated PII key is flagged",
			source: labelSource(`"api-key", apiKey`),
			expect: 1,
		},
		{
			name:   "a mixed-case PII key is flagged",
			source: labelSource(`"Password", password`),
			expect: 1,
		},
		{
			name:   "two PII keys in one call are both flagged",
			source: labelSource(`"user_id", userID, "email", email`),
			expect: 2,
		},
		{
			name:       "a backquoted raw-string PII key is flagged",
			source:     labelSource("`email`, email"),
			expect:     1,
			wantSubstr: `"email"`,
		},
		{
			name:       "an odd argument count still checks the dangling key-position arg",
			source:     labelSource(`"email"`),
			expect:     1,
			wantSubstr: `"email"`,
		},
	}
}

func checkFileCleanCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "no results has nothing to inspect",
			source: "package foo\nfunc run() {}\n",
			expect: 0,
		},
		{
			name: "zero arguments has nothing to check",
			source: "package foo\n" +
				"import \"runtime/pprof\"\n" +
				"func run() {\n" +
				"\tpprof.Labels()\n" +
				"}\n",
			expect: 0,
		},
		{
			name:   "a non-PII key is clean",
			source: labelSource(`"request_id", requestID`),
			expect: 0,
		},
		{
			name: "a non-literal key is not guessed at",
			source: "package foo\n" +
				"import \"runtime/pprof\"\n" +
				"func run(key, email string) {\n" +
				"\tpprof.Labels(key, email)\n" +
				"}\n",
			expect: 0,
		},
		{
			name:   "a PII word in the value position is ignored",
			source: labelSource(`"request_id", "email"`),
			expect: 0,
		},
		{
			name: "a differently named Labels call is not pprof.Labels",
			source: "package foo\n" +
				"type builder struct{}\n" +
				"func (b builder) Labels(key, value string) {}\n" +
				"func run(b builder) { b.Labels(\"email\", \"a@b.com\") }\n",
			expect: 0,
		},
	}
}

func pprofImportScopeCases() []checkFileTestCase {
	return append(pprofUnflaggedImportCases(), pprofFlaggedImportCases()...)
}

func pprofUnflaggedImportCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a shadowed local named pprof with no import is never flagged",
			source: "package foo\n" +
				"type notPprof struct{}\n" +
				"func (notPprof) Labels(pairs ...string) []string { return pairs }\n" +
				"func run() []string {\n" +
				"\tpprof := notPprof{}\n" +
				"\treturn pprof.Labels(\"user_id\", \"x\")\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "a local pprof shadowing the import in the same function is not flagged",
			source: "package foo\n" +
				"import \"runtime/pprof\"\n" +
				"type notPprof struct{}\n" +
				"func (notPprof) Labels(pairs ...string) []string { return pairs }\n" +
				"func run() []string {\n" +
				"\tpprof := notPprof{}\n" +
				"\treturn pprof.Labels(\"user_id\", \"x\")\n" +
				"}\n" +
				"func g(email string) pprof.LabelSet { return pprof.Labels(\"email\", email) }\n",
			expect:     1,
			wantSubstr: `"email"`,
		},
		{
			name: "an imported file with a differently named Labels call is not pprof.Labels",
			source: "package foo\n" +
				"import \"runtime/pprof\"\n" +
				"type builder struct{}\n" +
				"func (b builder) Labels(key, value string) {}\n" +
				"func run(b builder) { b.Labels(\"email\", \"a@b.com\") }\n" +
				"var _ = pprof.Labels\n",
			expect: 0,
		},
	}
}

func pprofFlaggedImportCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a real pprof.Labels call with the proper import is flagged",
			source: "package foo\n" +
				"import \"runtime/pprof\"\n" +
				"func run(userID string) pprof.LabelSet {\n" +
				"\treturn pprof.Labels(\"user_id\", userID)\n" +
				"}\n",
			expect:     1,
			wantSubstr: `"user_id"`,
		},
		{
			name: "an aliased runtime/pprof import is still matched by its local name",
			source: "package foo\n" +
				"import p \"runtime/pprof\"\n" +
				"func run(userID string) p.LabelSet {\n" +
				"\treturn p.Labels(\"user_id\", userID)\n" +
				"}\n",
			expect:     1,
			wantSubstr: `"user_id"`,
		},
		{
			name: "an aliased import is not matched by the default pprof name",
			source: "package foo\n" +
				"import p \"runtime/pprof\"\n" +
				"func run(pprof, userID string) p.LabelSet {\n" +
				"\t_ = pprof\n" +
				"\treturn p.Labels(\"user_id\", userID)\n" +
				"}\n",
			expect:     1,
			wantSubstr: `"user_id"`,
		},
	}
}

func labelSource(args string) string {
	return "package foo\n" +
		"import \"runtime/pprof\"\n" +
		"func run(email, userID, apiKey, password, requestID string) {\n" +
		"\tpprof.Labels(" + args + ")\n" +
		"}\n"
}

func analyzeSource(t *testing.T, source string) []core.Finding {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), fsperm.File); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	cache := astcache.New()
	file, err := cache.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	fc := core.FileChange{Path: "foo.go", Extension: ".go"}
	return checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter)
}
