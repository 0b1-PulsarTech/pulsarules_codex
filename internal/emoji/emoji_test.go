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

// The catalog is generated data, so its shape is asserted here rather than
// trusted: a truncated or corrupt data file would otherwise silently accept or
// reject every commit.
const (
	wantCatalogSize = 1054
	wantPoolSize    = 728
)

// mustCatalog builds the real embedded catalog or fails the test, so every
// test in this package can assume a valid *Catalog without repeating the
// error check.
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

	catalog := mustCatalog(t)
	if got := catalog.Size(); got != wantCatalogSize {
		t.Fatalf("Size() = %d, want %d", got, wantCatalogSize)
	}
	if got := catalog.PoolSize(); got != wantPoolSize {
		t.Fatalf("PoolSize() = %d, want %d", got, wantPoolSize)
	}
	if catalog.PoolSize() >= catalog.Size() {
		t.Fatalf("pool %d must be narrower than catalog %d", catalog.PoolSize(), catalog.Size())
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

// The suggestion pool must stay clear of emoji that carry no engineering
// meaning, even though the catalog allows them.
func TestCatalogPoolMembership(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		shortcode string
		want      bool
	}{
		{"object stays", "wrench", true},
		{"animal stays", "boar", true},
		{"food stays", "tea", true},
		{"travel stays", "sailboat", true},
		{"non-country flag stays", "checkered_flag", true},
		{"monkey face stays", "see_no_evil", true},
		{"unused monkey face stays", "speak_no_evil", true},
		{"measured usage outranks its group", "skull", true},
		{"initial-commit emoji stays", "ghost", true},
		{"zodiac drops", "aquarius", false},
		{"arrow drops", "arrow_lower_left", false},
		{"letters drop", "abcd", false},
		{"face drops", "astonished", false},
	}

	catalog := mustCatalog(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if !catalog.Allows(testCase.shortcode) && testCase.want {
				t.Fatalf("%q must be in the catalog to be in the pool", testCase.shortcode)
			}
			if got := catalog.inPool[testCase.shortcode]; got != testCase.want {
				t.Fatalf("inPool[%q] = %v, want %v", testCase.shortcode, got, testCase.want)
			}
		})
	}
}

func TestCatalogPoolIsSubsetOfAllowed(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	for _, shortcode := range catalog.pool {
		if !catalog.Allows(shortcode) {
			t.Fatalf("pool entry %q is not allowed", shortcode)
		}
	}
}

// The type buckets only bias suggestions, so every entry must be recommendable.
func TestCatalogTypeBucketsAreAllowed(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	if len(catalog.byType) == 0 {
		t.Fatal("byType is empty; the generated type data did not load")
	}
	for commitType, shortcodes := range catalog.byType {
		if len(shortcodes) == 0 {
			t.Fatalf("type %q has no shortcodes", commitType)
		}
		for _, shortcode := range shortcodes {
			if !catalog.Allows(shortcode) {
				t.Fatalf("type %q lists disallowed shortcode %q", commitType, shortcode)
			}
		}
	}
}

// The pool is ordered by measured usage so the head of a suggestion set
// carries the strongest signal.
func TestCatalogPoolIsRankedByUsage(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	previous := catalog.allowed[catalog.pool[0]].janeCount
	for _, shortcode := range catalog.pool[1:] {
		count := catalog.allowed[shortcode].janeCount
		if count > previous {
			t.Fatalf("pool unranked: %q (%d) follows count %d", shortcode, count, previous)
		}
		previous = count
	}
	if !slices.Contains(catalog.pool[:5], "wrench") {
		t.Fatalf("expected the most used shortcode near the head, got %v", catalog.pool[:5])
	}
}

// TestReadRecords exercises the malformed-input paths against a fabricated
// fs.FS rather than corrupting the real embedded data: a short or overlong
// row is a tools/emojigen build defect, and it must fail loudly naming the
// file instead of collapsing the whole catalog to empty.
func TestReadRecords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		fsys       fs.FS
		file       string
		fieldCount int
		want       [][]string
		wantErr    error
	}{
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
			for i, row := range got {
				if !slices.Equal(row, testCase.want[i]) {
					t.Fatalf("records = %v, want %v", got, testCase.want)
				}
			}
		})
	}
}

// TestReadCatalogRejectsUnparseableCount pins NewCatalog's own promise: a malformed generator row
// is reported, never silently dropped. A dropped shortcode is invisible here and then surfaces far
// away, as commitlint rejecting a commit emoji that the catalog file plainly contains.
func TestReadCatalogRejectsUnparseableCount(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{"data/catalog.txt": &fstest.MapFile{
		Data: []byte(":sparkles:\tActivities\tevent\tnot-a-number\n"),
	}}
	catalog := &Catalog{allowed: map[string]entry{}, byType: map[string][]string{}}

	err := catalog.readCatalog(fsys)
	if err == nil {
		t.Fatalf(
			"readCatalog returned nil; catalog silently holds %d entries",
			len(catalog.allowed),
		)
	}
	if !strings.Contains(err.Error(), ":sparkles:") {
		t.Errorf("error = %q, want it to name the offending shortcode", err)
	}
}
