package governance

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
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
