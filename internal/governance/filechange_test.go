package governance

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/testfile"
)

func TestChangedSources(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		files []pass.FileChange
		want  []syntax.Source
	}{
		{name: "no files", want: []syntax.Source{}},
		{
			name: "test flag is carried and order kept",
			files: []pass.FileChange{
				{Path: "b.go", IsTest: true, Staged: true},
				{Path: "a.go"},
				{Path: "c.go"},
			},
			want: []syntax.Source{
				{Path: "b.go", IsTest: true},
				{Path: "a.go"},
				{Path: "c.go"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := changedSources(testCase.files); !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("changedSources() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestWalkAllFilesSetsIsTest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
		want bool
	}{
		{"go test suffix", "a_test.go", true},
		{"go source", "a.go", false},
		{"go helper under tests dir", "tests/internal/boot.go", false},
		{"mjs under test dir", "web/test/ajustes.mjs", true},
		{"dart helper under test dir", "packages/x/test/helper.dart", true},
		{"ts name with test prefix only", "src/test_utils.ts", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			files := fstest.MapFS{testCase.path: &fstest.MapFile{}}
			provider := source.NewFSProvider(files, "", source.SkipSet{})
			got := discovery.WalkAllFiles(provider, testfile.Defaults())
			if len(got) != 1 || got[0].Path != testCase.path || got[0].IsTest != testCase.want {
				t.Fatalf("walked %+v, want %q with IsTest %v", got, testCase.path, testCase.want)
			}
		})
	}
}

func TestChangesToFileChangesSetsIsTest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		change vcs.Change
		want   pass.FileChange
	}{
		{
			name:   "go test suffix",
			change: vcs.Change{Path: "a_test.go", Extension: ".go", Staged: true},
			want: pass.FileChange{
				Path: "a_test.go", Extension: ".go", IsTest: true, Staged: true,
			},
		},
		{
			name:   "mjs under test dir",
			change: vcs.Change{Path: "web/test/a.mjs", Extension: ".mjs"},
			want:   pass.FileChange{Path: "web/test/a.mjs", Extension: ".mjs", IsTest: true},
		},
		{
			name:   "go helper under tests dir",
			change: vcs.Change{Path: "tests/boot.go", Extension: ".go"},
			want:   pass.FileChange{Path: "tests/boot.go", Extension: ".go"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := discovery.ChangesToFileChanges(
				[]vcs.Change{testCase.change},
				testfile.Defaults(),
			)
			if len(got) != 1 || got[0] != testCase.want {
				t.Fatalf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}

type renamesFailingRepo struct{ *goldenRepo }

func (renamesFailingRepo) StagedRenames(int) ([]vcs.Rename, error) {
	return nil, errors.New("rename probe exploded")
}

func TestSession_StagedRenamesLogsProbeFailure(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	sess := NewSession(
		renamesFailingRepo{&goldenRepo{root: t.TempDir()}},
		nil,
		testCatalog(t),
		nil,
	)
	sess.logger = slog.New(slog.NewTextHandler(&logs, nil))

	if renames := sess.stagedRenames(); len(renames) != 0 {
		t.Fatalf("stagedRenames() = %v, want none", renames)
	}
	if !strings.Contains(logs.String(), "rename probe exploded") {
		t.Fatalf("log output %q does not report the probe failure", logs.String())
	}
}
