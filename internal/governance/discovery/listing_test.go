package discovery

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type fakeLister struct {
	root  string
	paths []string
	err   error
}

func (lister fakeLister) Root() string { return lister.root }

func (lister fakeLister) ListFiles() ([]string, error) { return lister.paths, lister.err }

func seedTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, rel := range []string{"a.go", "ignored.go", "vendor/v.go", "other/o.go"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("package x\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return root
}

func walked(provider source.Provider) []string {
	var paths []string
	provider.Walk(func(path, _ string) bool {
		paths = append(paths, path)
		return true
	})
	slices.Sort(paths)
	return paths
}

func TestListed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		lister      func(root string) fakeLister
		wantWalk    []string
		wantSkipped string
		wantLog     string
	}{
		{
			name: "git's list is walked and only the git skips apply",
			lister: func(root string) fakeLister {
				return fakeLister{root: root, paths: []string{"a.go", "vendor/v.go", "other/o.go"}}
			},
			wantWalk:    []string{"a.go", "other/o.go"},
			wantSkipped: "vendor/v.go",
		},
		{
			name: "a failed listing falls back to a filesystem walk with the plain skips",
			lister: func(root string) fakeLister {
				return fakeLister{root: root, err: errors.New("git unavailable")}
			},
			wantWalk:    []string{"a.go", "ignored.go"},
			wantSkipped: "other/o.go",
			wantLog:     "walking the filesystem instead of git's file list",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			skips := source.NewSkipSet([]string{"vendor", "other"})
			gitSkips := source.NewSkipSet([]string{"vendor"})

			provider, applied := Listed(testCase.lister(seedTree(t)), skips, gitSkips, logger)
			if got := walked(provider); !slices.Equal(got, testCase.wantWalk) {
				t.Fatalf("walked %v, want %v", got, testCase.wantWalk)
			}
			if !applied.SkipsPath(testCase.wantSkipped) {
				t.Errorf("returned skip set does not skip %q", testCase.wantSkipped)
			}
			if !strings.Contains(logs.String(), testCase.wantLog) {
				t.Errorf("log %q lacks %q", logs.String(), testCase.wantLog)
			}
		})
	}
}
