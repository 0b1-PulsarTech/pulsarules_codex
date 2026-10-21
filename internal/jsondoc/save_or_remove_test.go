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
			if err = doc.SaveOrRemove(); err != nil {
				t.Fatalf("SaveOrRemove: %v", err)
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
	if err = missing.SaveOrRemove(); err == nil {
		t.Fatal("SaveOrRemove(empty, no file) = nil, want a remove error")
	}

	orphan, err := Load(filepath.Join(t.TempDir(), "no-parent", "config.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = SetSection(orphan, "a", 1); err != nil {
		t.Fatalf("SetSection: %v", err)
	}
	if err = orphan.SaveOrRemove(); err == nil {
		t.Fatal("SaveOrRemove(no parent dir) = nil, want a save error")
	}
}
