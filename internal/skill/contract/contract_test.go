package contract

import (
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestSession_EmbeddedTemplates(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	session, err := Session(templates)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if session == "" {
		t.Fatal("Session returned empty text")
	}
}

func TestSession_MissingAsset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		templates fstest.MapFS
	}{
		{
			name:      "Session missing contract.txt",
			templates: fstest.MapFS{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Session(testCase.templates); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}
