package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

func TestWriteCatalogAndTypes_PlaceFieldsAtTheReaderColumns(t *testing.T) {
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
		name string
		path string
		want [][]string
	}{
		{"catalog", catalogPath, [][]string{{"wrench", "Objects", "tool", "3"}}},
		{"types", typesPath, [][]string{{"build", "wrench", "5"}, {"fix", "wrench", "2"}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := readFields(t, testCase.path)
			if len(got) != len(testCase.want) {
				t.Fatalf("rows = %v, want %v", got, testCase.want)
			}
			for rowIdx, want := range testCase.want {
				if strings.Join(got[rowIdx], "\t") != strings.Join(want, "\t") {
					t.Errorf("row %d = %v, want %v", rowIdx, got[rowIdx], want)
				}
			}
		})
	}
}

func readFields(t *testing.T, path string) [][]string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var rows [][]string
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		if !strings.HasPrefix(line, "#") {
			rows = append(rows, strings.Split(line, "\t"))
		}
	}
	return rows
}

func TestWriteCatalogAndTypes_ReproduceTheCommittedData(t *testing.T) {
	t.Parallel()

	dataDir := filepath.Join("..", "..", "internal", "emoji", "data")
	entries := make([]catalogEntry, 0, 1100)
	allowed := make(map[string]bool)
	for _, fields := range readFields(t, filepath.Join(dataDir, "catalog.txt")) {
		count, err := strconv.Atoi(fields[emoji.CatalogUseCount])
		if err != nil {
			t.Fatalf("use count %q: %v", fields[emoji.CatalogUseCount], err)
		}
		entries = append(entries, catalogEntry{
			Name:     fields[emoji.CatalogName],
			Group:    fields[emoji.CatalogGroup],
			Subgroup: fields[emoji.CatalogSubgroup],
			UseCount: count,
		})
		allowed[fields[emoji.CatalogName]] = true
	}
	byType := make(map[string]map[string]int)
	for _, fields := range readFields(t, filepath.Join(dataDir, "types.txt")) {
		count, err := strconv.Atoi(fields[emoji.TypesCount])
		if err != nil {
			t.Fatalf("count %q: %v", fields[emoji.TypesCount], err)
		}
		if byType[fields[emoji.TypesType]] == nil {
			byType[fields[emoji.TypesType]] = make(map[string]int)
		}
		byType[fields[emoji.TypesType]][fields[emoji.TypesName]] = count
	}
	out := t.TempDir()
	if err := writeCatalog(filepath.Join(out, "catalog.txt"), entries); err != nil {
		t.Fatal(err)
	}
	if err := writeTypes(filepath.Join(out, "types.txt"), byType, allowed); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"catalog.txt", "types.txt"} {
		want, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("regenerated %s differs from the committed file the reader loads", name)
		}
	}
}

func TestWriteTable_UnwritablePathFails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no", "such", "dir.txt")
	if err := writeTable(missing, "# header\n", nil); err == nil {
		t.Fatal("writeTable into a missing directory returned nil")
	}
}
