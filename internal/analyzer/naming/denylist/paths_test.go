package denylist

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func presentFiles() fstest.MapFS {
	return fstest.MapFS{
		"src/draft/a.dart":          {},
		"src/draft/b.dart":          {},
		"src/legacy_view.dart":      {},
		"draft/inner/legacy/a.dart": {},
		"main.dart":                 {},
		"draft/a.dart":              {},
		"src/drafts/a.dart":         {},
		"lib/view.dart":             {},
		"lib/pubspec.lock":          {},
		"lib/.legacy":               {},
		"build/legacy/a.dart":       {},
	}
}

func pathSettings() settings {
	return settings{
		reporter:    finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
		dirReporter: finding.NewReporter(dirID, finding.SeverityWarning, finding.CatAST),
		words: newMatcher(
			[]string{"draft", "drafts", "legacy", "dart", "lock"}, []string{"drafts"},
		),
	}
}

func TestCheckPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		changed []pass.FileChange
		want    []string
	}{
		{
			"two files in one directory give one directory finding",
			[]pass.FileChange{{Path: "src/draft/a.dart"}, {Path: "src/draft/b.dart"}},
			[]string{"forbidden-token/dir src/draft"},
		},
		{
			"file name",
			[]pass.FileChange{{Path: "src/legacy_view.dart"}},
			[]string{"forbidden-token src/legacy_view.dart"},
		},
		{
			"nested directories report their own prefix",
			[]pass.FileChange{{Path: "draft/inner/legacy/a.dart"}},
			[]string{"forbidden-token/dir draft", "forbidden-token/dir draft/inner/legacy"},
		},
		{"top level file has no directory", []pass.FileChange{{Path: "main.dart"}}, nil},
		{
			"generated file is skipped",
			[]pass.FileChange{{Path: "draft/a.dart", Generated: true}},
			nil,
		},
		{"allowed segment", []pass.FileChange{{Path: "src/drafts/a.dart"}}, nil},
		{
			"extension is not part of the file name",
			[]pass.FileChange{{Path: "lib/view.dart"}, {Path: "lib/pubspec.lock"}},
			nil,
		},
		{
			"dotfile keeps its full name",
			[]pass.FileChange{{Path: "lib/.legacy"}},
			[]string{"forbidden-token lib/.legacy"},
		},
		{
			"skipped directory is not checked",
			[]pass.FileChange{{Path: "build/legacy/a.dart"}},
			nil,
		},
		{
			"deleted file and its empty directory are not reported",
			[]pass.FileChange{{Path: "src/draft/legacy_gone.dart"}},
			nil,
		},
	}
	cfg := pathSettings()
	skips := source.NewSkipSet([]string{"build"})
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, hit := range checkPaths(&pass.Context{
				ChangedFiles: testCase.changed,
				Sources:      source.NewFSProvider(presentFiles(), "", skips),
				Skips:        skips,
			}, cfg) {
				if hit.Line != 0 {
					t.Fatalf("path finding line = %d, want 0", hit.Line)
				}
				got = append(got, hit.AnalyzerID+" "+hit.File)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("findings = %q, want %q", got, testCase.want)
			}
		})
	}
}
