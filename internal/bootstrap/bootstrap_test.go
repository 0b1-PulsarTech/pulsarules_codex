package bootstrap

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/logfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestNew_ResolvesEveryService(t *testing.T) {
	t.Parallel()

	svc, err := New(Options{ProjectDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	testCases := []struct {
		name string
		set  bool
	}{
		{"knowledge index", svc.Index != nil},
		{"templates", svc.Templates.FS != nil},
		{"renderer", svc.Renderer != nil},
		{"config loader", svc.Loader != nil},
		{"target registry", svc.Targets != nil},
		{"open repository", svc.OpenRepository != nil},
		{"open logger", svc.OpenLogger != nil},
		{"load catalog", svc.LoadCatalog != nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if !testCase.set {
				t.Fatalf("%s is not wired", testCase.name)
			}
		})
	}
	if _, err = svc.LoadCatalog(); err != nil {
		t.Errorf("LoadCatalog: %v", err)
	}
	logger, closer, err := svc.OpenLogger()
	if err != nil || logger == nil || closer == nil {
		t.Errorf("OpenLogger() = %v, %v, %v, want a logger and a closer", logger, closer, err)
	}
	if repo, repoErr := svc.OpenRepository(); repoErr != nil || repo == nil {
		t.Errorf(
			"OpenRepository() = %v, %v, want the repository of the working tree",
			repo,
			repoErr,
		)
	}
}

func TestNew_KnowledgeLoadFailure(t *testing.T) {
	t.Parallel()

	_, err := New(Options{Root: t.TempDir()})
	if err == nil {
		t.Fatal("New: want error for a dev root with no knowledge directory, got nil")
	}
}

func TestNewServices_BuildsOnlyWhatIsAsked(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "x.log")
	counts := &buildCounts{}
	counted := counts.builders()

	userDir := t.TempDir()
	svc, err := newServices(Options{
		ProjectDir: ".", LogLevel: "debug", LogPath: logPath, UserConfigDir: userDir,
	}, counted)
	if err != nil {
		t.Fatalf("newServices: %v", err)
	}
	if _, statErr := os.Stat(logPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf(
			"the log file exists after New (stat err = %v), want it created on first use",
			statErr,
		)
	}

	testCases := []struct {
		name  string
		ask   func()
		built *int
	}{
		{"repository", func() { _, _ = svc.OpenRepository() }, &counts.repositories},
		{"catalog", func() { _, _ = svc.LoadCatalog() }, &counts.catalogs},
		{"config loader", func() { _ = svc.Loader() }, &counts.loaders},
		{"logger", func() {
			_, closer, _ := svc.OpenLogger()
			t.Cleanup(func() { _ = closer.Close() })
		}, &counts.loggers},
	}
	for _, testCase := range testCases {
		if *testCase.built != 0 {
			t.Fatalf("%s was built by New, want it built on first use", testCase.name)
		}
	}
	for _, testCase := range testCases {
		testCase.ask()
		if *testCase.built != 1 {
			t.Errorf("%s builds after one request = %d, want 1", testCase.name, *testCase.built)
		}
	}
	requireUserDir(t, counts.loaderDir, userDir)
	if _, statErr := os.Stat(logPath); statErr != nil {
		t.Errorf("the log file is missing after the logger was asked for: %v", statErr)
	}
	for _, testCase := range testCases[1:] {
		testCase.ask()
		if *testCase.built != 1 {
			t.Errorf(
				"%s was rebuilt on a second request (%d builds)",
				testCase.name,
				*testCase.built,
			)
		}
	}
}

func requireUserDir(t *testing.T, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("the loader got UserConfigDir %q, want %q from Options", got, want)
	}
}

type buildCounts struct {
	repositories, catalogs, loaders, loggers int
	loaderDir                                string
}

func (counts *buildCounts) builders() builders {
	counted := defaultBuilders()
	openRepository, newCatalog, newLoader, newLogger :=
		counted.openRepository, counted.newCatalog, counted.newLoader, counted.newLogger
	counted.openRepository = func(dir string) (*vcs.Repository, error) {
		counts.repositories++
		return openRepository(dir)
	}
	counted.newCatalog = func() (*emoji.Catalog, error) {
		counts.catalogs++
		return newCatalog()
	}
	counted.newLoader = func(userConfigDir string) *loader.Layered {
		counts.loaders++
		counts.loaderDir = userConfigDir
		return newLoader(userConfigDir)
	}
	counted.newLogger = func(cfg logfile.Config) (*slog.Logger, io.Closer, error) {
		counts.loggers++
		return newLogger(cfg)
	}
	return counted
}
