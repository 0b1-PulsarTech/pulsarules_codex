package jsondoc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocument_SaveOrRemove(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		content     string
		mutate      func(doc *Document)
		wantRemoved bool
	}{
		{
			name:        "empty document removes the file",
			content:     `{"only":1}`,
			mutate:      func(doc *Document) { doc.Delete("only") },
			wantRemoved: true,
		},
		{
			name:    "non-empty document is saved",
			content: `{"only":1,"other":2}`,
			mutate:  func(doc *Document) { doc.Delete("only") },
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := seed(t, testCase.content)
			doc, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			testCase.mutate(doc)
			removed, err := doc.SaveOrRemove()
			if err != nil {
				t.Fatalf("SaveOrRemove: %v", err)
			}
			if removed != testCase.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}
			_, statErr := os.Stat(path)
			if (statErr == nil) == testCase.wantRemoved {
				t.Errorf("file exists = %v, want %v", statErr == nil, !testCase.wantRemoved)
			}
		})
	}
}

func TestDocument_SaveOrRemove_Failures(t *testing.T) {
	t.Parallel()

	missing, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	removed, err := missing.SaveOrRemove()
	if err == nil || removed {
		t.Fatalf("SaveOrRemove(empty, no file) = %v, %v, want a remove error", removed, err)
	}

	orphan, err := Load(filepath.Join(t.TempDir(), "no-parent", "config.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = SetSection(orphan, "a", 1); err != nil {
		t.Fatalf("SetSection: %v", err)
	}
	removed, err = orphan.SaveOrRemove()
	if err == nil || removed {
		t.Fatalf("SaveOrRemove(no parent dir) = %v, %v, want a save error", removed, err)
	}
}
