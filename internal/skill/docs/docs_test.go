package docs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func loadFixture(tb testing.TB) (*knowledge.Index, *render.Renderer) {
	tb.Helper()
	idx, templates, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("Load: %v", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		tb.Fatalf("NewRenderer: %v", err)
	}
	return idx, rnd
}

func seedFile(tb testing.TB, filePath, content string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", filePath, err)
	}
}

func mustWriteDoc(tb testing.TB, dir, body string) {
	tb.Helper()
	if _, err := Write(dir, "SKILL.md", body); err != nil {
		tb.Fatalf("WriteDoc: %v", err)
	}
}

func assertPresent(tb testing.TB, filePath string) {
	tb.Helper()
	if _, err := os.Stat(filePath); err != nil {
		tb.Errorf("expected %q to exist, stat err = %v", filePath, err)
	}
}

func assertMissing(tb testing.TB, filePath string) {
	tb.Helper()
	if _, err := os.Stat(filePath); !errors.Is(err, fs.ErrNotExist) {
		tb.Errorf("expected %q to be absent, stat err = %v", filePath, err)
	}
}

func assertContent(tb testing.TB, filePath, want string) {
	tb.Helper()
	got, err := os.ReadFile(filePath) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read %q: %v", filePath, err)
	}
	if string(got) != want {
		tb.Errorf("%q content = %q, want %q", filePath, got, want)
	}
}

func assertIDs(tb testing.TB, label string, got []string, want ...string) {
	tb.Helper()
	if !slices.Equal(got, want) {
		tb.Errorf("%s = %v, want %v", label, got, want)
	}
}
