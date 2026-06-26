package knowledge

import (
	"strings"
	"testing"
)

func TestLoad_EmbeddedCountsAndTransclusionBodies(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	testCases := []struct {
		name      string
		got, want int
	}{
		{"rules", len(idx.Rules), 32},
		{"patterns", len(idx.Patterns), 25},
		{"workflows", len(idx.Workflows), 8},
		{"skills", len(idx.Skills), 29},
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

func TestLoad_BadRoot(t *testing.T) {
	t.Parallel()

	if _, _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected error loading from an empty root, got nil")
	}
}

func TestLoad_EveryWorkflowHasSteps(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, workflow := range idx.Workflows {
		if len(workflow.Steps) == 0 {
			t.Errorf("workflow %q has no steps", workflow.ID)
		}
	}
}
