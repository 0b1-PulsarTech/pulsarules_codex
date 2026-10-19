package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

func TestWriteCatalogAndTypes_MatchTheSharedRowFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.txt")
	typesPath := filepath.Join(dir, "types.txt")
	entries := []catalogEntry{{Name: "wrench", Group: "Objects", Subgroup: "tool", UseCount: 3}}
	byType := map[string]map[string]int{
		"fix":   {"wrench": 2, "gone": 9},
		"build": {"wrench": 5},
	}
	if err := writeCatalog(catalogPath, entries); err != nil {
		t.Fatalf("writeCatalog: %v", err)
	}
	if err := writeTypes(typesPath, byType, map[string]bool{"wrench": true}); err != nil {
		t.Fatalf("writeTypes: %v", err)
	}
	testCases := []struct {
		path string
		want string
	}{
		{catalogPath, emoji.CatalogHeader + "wrench\tObjects\ttool\t3\n"},
		{typesPath, emoji.TypesHeader + "build\twrench\t5\nfix\twrench\t2\n"},
	}
	for _, testCase := range testCases {
		got, err := os.ReadFile(testCase.path)
		if err != nil {
			t.Fatalf("read %s: %v", testCase.path, err)
		}
		if string(got) != testCase.want {
			t.Errorf("%s = %q, want %q", testCase.path, got, testCase.want)
		}
	}
}

func TestWriteTable_UnwritablePathFails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no", "such", "dir.txt")
	if err := writeTable(missing, emoji.TypesHeader, nil); err == nil {
		t.Fatal("writeTable into a missing directory returned nil")
	}
}
