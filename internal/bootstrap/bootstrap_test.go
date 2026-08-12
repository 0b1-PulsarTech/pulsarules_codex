package bootstrap

import (
	"io"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// why: one row per DoInjections binding, named for the failure message.
type resolveEveryBinding struct {
	name   string
	lookup func(remy.Injector) error
}

//nolint:tparallel // subtests deliberately run sequentially; see the comment below.
func TestDoInjections_ResolvesEveryBinding(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := DoInjections(inj, Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	testCases := []resolveEveryBinding{
		{"knowledge index", func(i remy.Injector) error {
			_, err := remy.Get[*knowledge.Index](i)
			return err
		}},
		{"templates fs.FS", func(i remy.Injector) error {
			_, err := remy.Get[fs.FS](i)
			return err
		}},
		{"renderer", func(i remy.Injector) error {
			_, err := remy.Get[*render.Renderer](i)
			return err
		}},
		{"vcs repository", func(i remy.Injector) error {
			_, err := remy.Get[vcs.Repository](i)
			return err
		}},
		{"logger", func(i remy.Injector) error {
			_, err := remy.Get[*slog.Logger](i)
			return err
		}},
		{"logger closer", func(i remy.Injector) error {
			_, err := remy.Get[io.Closer](i)
			return err
		}},
		{"install registry", func(i remy.Injector) error {
			_, err := remy.Get[*install.Registry](i)
			return err
		}},
		{"target registry", func(i remy.Injector) error {
			_, err := remy.Get[*target.Registry](i)
			return err
		}},
	}
	// why: subtests share the one injector built above, and remy's own docs
	// call concurrent access to a shared Injector unsupported, so these run
	// sequentially rather than with the usual inner t.Parallel().
	for _, testCase := range testCases { //nolint:paralleltest // see comment above.
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
