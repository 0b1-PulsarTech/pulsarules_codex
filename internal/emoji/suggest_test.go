package emoji

import (
	"slices"
	"testing"
)

func TestSuggestCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		commitType string
		count      int
		want       int
	}{
		{"seven for a rich type", "feat", 7, 7},
		{"seven for a thin type", "revert", 7, 7},
		{"seven for an unknown type", "nonsense", 7, 7},
		{"three when three asked", "fix", 3, 3},
		{"one when one asked", "docs", 1, 1},
		{"none for zero", "feat", 0, 0},
		{"none for negative", "feat", -1, 0},
	}

	catalog := mustCatalog(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			picks := catalog.Suggest(testCase.commitType, "seed", nil, testCase.count)
			if len(picks) != testCase.want {
				t.Fatalf("got %d picks %v, want %d", len(picks), picks, testCase.want)
			}
		})
	}
}

// A suggestion must be actionable: in the pool, never banned, and never one of
// the emoji the window already rules out.
func TestSuggestOffersOnlyUsableShortcodes(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	excluded := []string{"wrench", "tea", "boar", "ram", "file_folder"}

	for _, commitType := range []string{"feat", "fix", "docs", "perf", "style", "revert", "chore"} {
		t.Run(commitType, func(t *testing.T) {
			t.Parallel()

			picks := catalog.Suggest(commitType, "some subject", excluded, 7)
			for _, pick := range picks {
				if slices.Contains(excluded, pick) {
					t.Fatalf("Suggest offered excluded shortcode %q", pick)
				}
				if IsProhibited(pick) || IsNonRendering(pick) {
					t.Fatalf("Suggest offered unusable shortcode %q", pick)
				}
				if !catalog.inPool[pick] {
					t.Fatalf("Suggest offered %q from outside the pool", pick)
				}
			}
			if len(picks) != len(slices.Compact(slices.Sorted(slices.Values(picks)))) {
				t.Fatalf("Suggest returned duplicates: %v", picks)
			}
		})
	}
}

func TestSuggestIsDeterministicPerSeed(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	first := catalog.Suggest("feat", "Add lead routing", nil, 7)
	second := catalog.Suggest("feat", "Add lead routing", nil, 7)

	if !slices.Equal(first, second) {
		t.Fatalf("same seed produced %v then %v", first, second)
	}
}

// Rotation is the whole point: a fixed suggestion set is what let runs of the
// same emoji build up in the first place.
func TestSuggestVariesBySeed(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	seeds := []string{
		"Add lead routing",
		"Fix decode error",
		"Rework the outbox relay",
		"Drop the legacy adapter",
	}

	seen := make(map[string]bool, len(seeds))
	for _, seed := range seeds {
		picks := catalog.Suggest("feat", seed, nil, 7)
		key := slices.Sorted(slices.Values(picks))[0]
		if seen[key] {
			continue
		}
		seen[key] = true
	}
	if len(seen) < 2 {
		t.Fatalf("suggestions did not vary across %d seeds", len(seeds))
	}
}

// The head of a set anchors on what the type actually uses, so the advice reads
// as relevant rather than random.
func TestSuggestAnchorsOnTheCommitType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		commitType string
	}{
		{"rich type", "feat"},
		{"thin docs type", "docs"},
		{"thin perf type", "perf"},
		{"thin test type", "test"},
	}

	catalog := mustCatalog(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			picks := catalog.Suggest(testCase.commitType, "a subject", nil, 7)
			bucket := catalog.byType[testCase.commitType]
			if !slices.Contains(bucket, picks[0]) {
				t.Fatalf("first pick %q is outside the %q vocabulary %v",
					picks[0], testCase.commitType, bucket)
			}
		})
	}
}

// An unknown type has no vocabulary to anchor on and must still produce advice.
func TestSuggestHandlesUnknownType(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	picks := catalog.Suggest("", "a subject", nil, 7)
	if len(picks) != 7 {
		t.Fatalf("Suggest with no type returned %v", picks)
	}
}

func TestDrawSpread(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		ranked    []string
		want      int
		wantLen   int
		wantFirst string
	}{
		{"empty input", nil, 3, 0, ""},
		{"fewer than asked", []string{"a", "b"}, 5, 2, "a"},
		{"exactly as asked", []string{"a", "b", "c"}, 3, 3, "a"},
		{"more than asked", []string{"a", "b", "c", "d", "e", "f"}, 3, 3, ""},
		{"zero wanted", []string{"a", "b"}, 0, 0, ""},
		{"negative wanted", []string{"a", "b"}, -1, 0, ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			picks := drawSpread(testCase.ranked, "seed", testCase.want)
			if len(picks) != testCase.wantLen {
				t.Fatalf("drawSpread returned %v, want %d entries", picks, testCase.wantLen)
			}
			if testCase.wantFirst != "" && picks[0] != testCase.wantFirst {
				t.Fatalf("first pick = %q, want %q", picks[0], testCase.wantFirst)
			}
		})
	}
}

// A band draw must never repeat an entry, or a suggestion set silently shrinks.
func TestDrawSpreadPicksDistinctEntries(t *testing.T) {
	t.Parallel()

	ranked := make([]string, 0, 40)
	for index := range 40 {
		ranked = append(ranked, string('a'+rune(index%26))+string('0'+rune(index/26)))
	}

	for _, seed := range []string{"one", "two", "three", "four", "five"} {
		picks := drawSpread(ranked, seed, 7)
		if len(slices.Compact(slices.Sorted(slices.Values(picks)))) != len(picks) {
			t.Fatalf("seed %q produced duplicates: %v", seed, picks)
		}
	}
}
