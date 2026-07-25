package hook

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestTypedChecklist(t *testing.T) {
	t.Parallel()

	t.Run("go and sql", func(t *testing.T) {
		t.Parallel()
		status := vcs.Status{Changes: []vcs.Change{
			{Path: "main.go", Extension: ".go"},
			{Path: "schema.sql", Extension: ".sql"},
		}}
		got := TypedChecklist(status)
		if !strings.Contains(got, "gofmt") {
			t.Fatalf("expected gofmt bullet, got: %s", got)
		}
		if !strings.Contains(got, "migrations reversible") {
			t.Fatalf("expected sql bullet, got: %s", got)
		}
	})

	t.Run("empty status produces empty checklist", func(t *testing.T) {
		t.Parallel()
		got := TypedChecklist(vcs.Status{})
		if got != "" {
			t.Fatalf("expected empty, got: %s", got)
		}
	})

	t.Run("no duplicate bullets", func(t *testing.T) {
		t.Parallel()
		status := vcs.Status{Changes: []vcs.Change{
			{Path: "a.go", Extension: ".go"},
			{Path: "b.go", Extension: ".go"},
		}}
		got := TypedChecklist(status)
		if strings.Count(got, "gofmt") != 1 {
			t.Fatalf("expected 1 gofmt bullet, got: %s", got)
		}
	})
}
