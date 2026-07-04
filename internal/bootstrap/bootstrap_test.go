package bootstrap

import (
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type resolveEveryBinding struct {
	name   string
	lookup func(remy.Injector) error
}

func TestDoInjections_ResolvesEveryBinding(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := doInjections(inj, Options{}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	testCases := []resolveEveryBinding{
		{"knowledge index", func(inj remy.Injector) error {
			_, err := remy.Get[*knowledge.Index](inj)
			return err
		}},
		{"renderer", func(inj remy.Injector) error {
			_, err := remy.Get[*render.Renderer](inj)
			return err
		}},
	}
	for _, testCase := range testCases {
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
	err := doInjections(inj, Options{Root: t.TempDir()})
	if err == nil {
		t.Fatal("DoInjections: want error for a dev root with no knowledge directory, got nil")
	}
}
