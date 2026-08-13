package emoji

import (
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

	const (
		wantCatalogSize = 1054
		wantPoolSize    = 728
	)
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

func TestCatalogPoolIsRankedByUsage(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	previous := catalog.allowed[catalog.pool[0]].useCount
	for _, shortcode := range catalog.pool[1:] {
		count := catalog.allowed[shortcode].useCount
		if count > previous {
			t.Fatalf("pool unranked: %q (%d) follows count %d", shortcode, count, previous)
		}
		previous = count
	}
	if !slices.Contains(catalog.pool[:5], "wrench") {
		t.Fatalf("expected the most used shortcode near the head, got %v", catalog.pool[:5])
	}
}

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
