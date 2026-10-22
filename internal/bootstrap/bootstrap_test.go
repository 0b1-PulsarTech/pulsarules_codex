package bootstrap

import (
	"io"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

type resolveEveryBinding struct {
	name   string
	lookup func(remy.Injector) error
}

//nolint:tparallel // Subtests share one injector, which remy does not support concurrently.
func TestDoInjections_ResolvesEveryBinding(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := DoInjections(inj, Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	testCases := []resolveEveryBinding{
		{"knowledge index", func(inj remy.Injector) error {
			_, err := remy.Get[*knowledge.Index](inj)
			return err
		}},
		{"templates fs.FS", func(inj remy.Injector) error {
			_, err := remy.Get[fs.FS](inj)
			return err
		}},
		{"templates set", func(inj remy.Injector) error {
			_, err := remy.Get[templates.Set](inj)
			return err
		}},
		{"renderer", func(inj remy.Injector) error {
			_, err := remy.Get[*render.Renderer](inj)
			return err
		}},
		{"vcs repository", func(inj remy.Injector) error {
			_, err := remy.Get[*vcs.Repository](inj)
			return err
		}},
		{"logger", func(inj remy.Injector) error {
			_, err := remy.Get[*slog.Logger](inj)
			return err
		}},
		{"logger closer", func(inj remy.Injector) error {
			_, err := remy.Get[io.Closer](inj)
			return err
		}},
		{"emoji catalog", func(inj remy.Injector) error {
			_, err := remy.Get[*emoji.Catalog](inj)
			return err
		}},
		{"target registry", func(inj remy.Injector) error {
			_, err := remy.Get[*host.Registry](inj)
			return err
		}},
	}
	for _, testCase := range testCases { //nolint:paralleltest // Shared injector.
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.lookup(inj); err != nil {
				t.Fatalf("resolve %s: %v", testCase.name, err)
			}
		})
	}
}

func TestDoInjections_KnowledgeLoadFailure(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := DoInjections(inj, Options{Root: t.TempDir()})
	if err == nil {
		t.Fatal("DoInjections: want error for a dev root with no knowledge directory, got nil")
	}
}

func TestNew_ResolvesEveryService(t *testing.T) {
	t.Parallel()

	svc, err := New(Options{ProjectDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eager := []struct {
		name string
		set  bool
	}{
		{"knowledge index", svc.Index != nil},
		{"templates", svc.Templates.FS != nil},
		{"renderer", svc.Renderer != nil},
		{"target registry", svc.Targets != nil},
		{"open repository", svc.OpenRepository != nil},
		{"open logger", svc.OpenLogger != nil},
		{"load catalog", svc.LoadCatalog != nil},
	}
	for _, testCase := range eager {
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
	logging, err := svc.OpenLogger()
	if err != nil || logging.Logger == nil || logging.Closer == nil {
		t.Errorf("OpenLogger() = %+v, %v, want a logger and a closer", logging, err)
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
