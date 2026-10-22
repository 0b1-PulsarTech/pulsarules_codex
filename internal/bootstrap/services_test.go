package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func TestResolve_MissingBinding(t *testing.T) {
	t.Parallel()

	_, err := resolve(remy.NewInjector(remy.Config{DuckTypeElements: true}), defaultBuilders())
	if err == nil || !strings.HasPrefix(err.Error(), "get knowledge index: ") {
		t.Fatalf("resolve() error = %v, want the knowledge index lookup to fail first", err)
	}
}

func TestServices_LazyFailuresStayLazy(t *testing.T) {
	t.Parallel()

	notADir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notADir, nil, disk.FilePrivate); err != nil {
		t.Fatalf("write file: %v", err)
	}
	testCases := []struct {
		name    string
		opts    Options
		open    func(Services) error
		wantErr string
	}{
		{
			name: "log dir under a file",
			opts: Options{
				ProjectDir: ".",
				LogLevel:   "debug",
				LogPath:    filepath.Join(notADir, "x.log"),
			},
			open: func(svc Services) error {
				_, _, err := svc.OpenLogger()
				return err
			},
			wantErr: "build logger: create log dir: ",
		},
		{
			name: "invalid log level",
			opts: Options{
				ProjectDir: ".",
				LogLevel:   "loud",
				LogPath:    filepath.Join(t.TempDir(), "x.log"),
			},
			open: func(svc Services) error {
				_, _, err := svc.OpenLogger()
				return err
			},
			wantErr: "build logger: parse log level: ",
		},
		{
			name: "project dir that is not a repository",
			opts: Options{ProjectDir: t.TempDir()},
			open: func(svc Services) error {
				_, err := svc.OpenRepository()
				return err
			},
			wantErr: "vcs: not a git repository",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			svc, err := New(testCase.opts)
			if err != nil {
				t.Fatalf("New must not build %s eagerly: %v", testCase.name, err)
			}
			if err = testCase.open(
				svc,
			); err == nil ||
				!strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("open error = %v, want containing %q", err, testCase.wantErr)
			}
		})
	}
}
