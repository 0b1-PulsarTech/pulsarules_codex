package knowledge

import (
	"strings"
	"testing"
)

// TestLoad_Embedded asserts the embedded snapshot loads with the expected counts
// and that bodies are captured for transclusion.
func TestLoad_Embedded(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	testCases := []struct {
		name      string
		got, want int
	}{
		{"rules", len(idx.Rules), 37},
		{"patterns", len(idx.Patterns), 29},
		{"workflows", len(idx.Workflows), 8},
		{"skills", len(idx.Skills), 36},
		{"references", len(idx.References), 14},
	}
	for _, testCase := range testCases {
		if testCase.got != testCase.want {
			t.Errorf("%s: got %d, want %d", testCase.name, testCase.got, testCase.want)
		}
	}

	if _, ok := idx.Rule("errors"); !ok {
		t.Error("missing errors rule")
	}
	if body := idx.Body("rules", "errors"); !strings.Contains(body, `{{define "must"}}`) {
		t.Errorf("errors body missing a section define, got %q", body[:min(len(body), 80)])
	}
}

// TestLoad_BadRoot asserts a non-existent disk root fails rather than silently
// falling back to the embedded snapshot.
func TestLoad_BadRoot(t *testing.T) {
	t.Parallel()

	if _, _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected error loading from an empty root, got nil")
	}
}
