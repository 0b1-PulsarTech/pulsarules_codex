package output

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// loadFixture loads the embedded index + renderer shared by the output tests.
func loadFixture(t testing.TB) (*knowledge.Index, *render.Renderer) {
	t.Helper()
	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	return idx, rnd
}
