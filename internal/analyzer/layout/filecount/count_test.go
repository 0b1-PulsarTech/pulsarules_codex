package filecount

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/testfile"
)

func TestCountByDirOnlyWantedDirs(t *testing.T) {
	t.Parallel()

	ctx := buildCtx(t, merged(numbered("a", 2), numbered("b", 3)), nil, nil)
	eligible := func(pass.FileChange) bool { return true }
	got := countByDir(ctx, eligible, map[string]bool{"a": true})
	if len(got) != 1 || got["a"].files != 2 {
		t.Fatalf("counts = %+v, want only a with 2 files", got)
	}
}

func TestSummarizeTieBreaksByName(t *testing.T) {
	t.Parallel()

	infos := []srcInfo{{pkg: "b"}, {pkg: "b"}, {pkg: "a", ent: true}, {pkg: "a", ent: true}}
	want := dirCount{files: 2, isEntSchema: true}
	for range 64 {
		if got := summarize(infos); got != want {
			t.Fatalf("summarize = %+v, want %+v", got, want)
		}
	}
}

type policyCase struct {
	name    string
	files   map[string]string
	changed []string
	globs   []string
	want    []string
}

func numberedTS(dir string, total int) map[string]string {
	files := make(map[string]string, total)
	for index := range total {
		files[fmt.Sprintf("%s/f%02d.ts", dir, index)] = "export {};\n"
	}
	return files
}

func policyCases() []policyCase {
	tsFlagged := []string{
		"web/src/f00.ts:1:package web/src has 11 non-test source files, max 10",
	}
	return []policyCase{
		{
			name: "ts files under a test dir are not counted",
			files: merged(
				numberedTS("web/src", 10), numberedTS("web/src/test", 3),
				map[string]string{"web/src/a.test.ts": "export {};\n"},
			),
			changed: []string{"web/src/f00.ts"},
		},
		{
			name:    "eleven plain ts files are flagged",
			files:   numberedTS("web/src", 11),
			changed: []string{"web/src/f00.ts"},
			want:    tsFlagged,
		},
		{
			name:    "test glob removes matching files from the count",
			files:   numberedTS("web/src", 11),
			changed: []string{"web/src/f00.ts"},
			globs:   []string{"web/src/f10.ts"},
		},
		{
			name:    "test glob with no match leaves the count",
			files:   numberedTS("web/src", 11),
			changed: []string{"web/src/f00.ts"},
			globs:   []string{"other/**"},
			want:    tsFlagged,
		},
	}
}

func TestAnalyzeTestPolicy(t *testing.T) {
	t.Parallel()

	tsParams := map[string]any{"extensions": []string{".ts"}}
	for _, testCase := range policyCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := buildCtx(t, testCase.files, testCase.changed, tsParams)
			classifier, err := testfile.New(testCase.globs)
			if err != nil {
				t.Fatalf("testfile.New: %v", err)
			}
			ctx.TestClassifier = classifier
			findings := NewAnalyzer().Analyze(ctx)
			got := make([]string, 0, len(findings))
			for _, finding := range findings {
				got = append(got, fmt.Sprintf(
					"%s:%d:%s", finding.File, finding.Line, finding.Message,
				))
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
