package emoji

import (
	"encoding/csv"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func mustCatalog(t *testing.T) *Catalog {
	t.Helper()
	catalog, err := NewCatalog()
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return catalog
}

func TestNewCatalogSize(t *testing.T) {
	t.Parallel()

	const wantCatalogSize = 1054
	catalog := mustCatalog(t)
	if got := catalog.Size(); got != wantCatalogSize {
		t.Fatalf("Size() = %d, want %d", got, wantCatalogSize)
	}
}

func TestCatalogAllows(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		shortcode string
		want      bool
	}{
		{"common object", "wrench", true},
		{"reference favourite", "tea", true},
		{"style favourite", "necktie", true},
		{"initial commit", "ghost", true},
		{"merge commit", "volcano", true},
		{"non-country flag", "checkered_flag", true},
		{"kept exception beetle", "beetle", true},
		{"kept exception egg", "egg", true},
		{"kept exception safety vest", "safety_vest", true},
		{"kept exception shopping cart", "shopping_cart", true},
		{"kept exception pirate flag", "pirate_flag", true},
		{"non-rendering clown", "clown_face", false},
		{"non-rendering bandage", "adhesive_bandage", false},
		{"non-rendering abacus", "abacus", false},
		{"prohibited robot", "robot", false},
		{"prohibited test tube", "test_tube", false},
		{"prohibited compass", "compass", false},
		{"prohibited sparkles", "sparkles", false},
		{"country flag us", "us", false},
		{"country flag cn", "cn", false},
		{"github proprietary", "octocat", false},
		{"unknown shortcode", "not_an_emoji_at_all", false},
	}

	catalog := mustCatalog(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := catalog.Allows(testCase.shortcode); got != testCase.want {
				t.Fatalf("Allows(%q) = %v, want %v", testCase.shortcode, got, testCase.want)
			}
		})
	}
}

type readRecordsCase struct {
	name       string
	fsys       fs.FS
	file       string
	fieldCount int
	want       [][]string
	wantErr    error
}

func readRecordsCases() []readRecordsCase {
	return []readRecordsCase{
		{
			name: "well-formed record set parses",
			fsys: fstest.MapFS{
				"catalog.txt": &fstest.MapFile{
					Data: []byte("wrench\tObjects\ttool\t3\ntea\tFood & Drink\tdrink\t1\n"),
				},
			},
			file:       "catalog.txt",
			fieldCount: 4,
			want: [][]string{
				{"wrench", "Objects", "tool", "3"},
				{"tea", "Food & Drink", "drink", "1"},
			},
		},
		{
			name: "too few fields is an error naming the file",
			fsys: fstest.MapFS{
				"catalog.txt": &fstest.MapFile{Data: []byte("wrench\tObjects\ttool\n")},
			},
			file:       "catalog.txt",
			fieldCount: 4,
			wantErr:    csv.ErrFieldCount,
		},
		{
			name: "too many fields is an error",
			fsys: fstest.MapFS{
				"catalog.txt": &fstest.MapFile{
					Data: []byte("wrench\tObjects\ttool\t3\textra\n"),
				},
			},
			file:       "catalog.txt",
			fieldCount: 4,
			wantErr:    csv.ErrFieldCount,
		},
		{
			name:       "missing file is an error",
			fsys:       fstest.MapFS{},
			file:       "catalog.txt",
			fieldCount: 4,
			wantErr:    fs.ErrNotExist,
		},
	}
}

func TestReadRecordsNamesTheFileOnMalformedRows(t *testing.T) {
	t.Parallel()

	testCases := readRecordsCases()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := readRecords(testCase.fsys, testCase.file, testCase.fieldCount)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("err = %v, want it to wrap %v", err, testCase.wantErr)
				}
				if !strings.Contains(err.Error(), testCase.file) {
					t.Fatalf("err = %v, want it to name %q", err, testCase.file)
				}
				return
			}
			if err != nil {
				t.Fatalf("readRecords: %v", err)
			}
			if len(got) != len(testCase.want) {
				t.Fatalf("records = %v, want %v", got, testCase.want)
			}
			for idx, row := range got {
				if !slices.Equal(row, testCase.want[idx]) {
					t.Fatalf("records = %v, want %v", got, testCase.want)
				}
			}
		})
	}
}
