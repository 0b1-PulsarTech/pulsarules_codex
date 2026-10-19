package emoji

import (
	"slices"
	"testing"
)

func TestThemeShortcodesAreSuggestible(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	for _, candidate := range catalog.themes {
		if len(candidate.keywords) == 0 || len(candidate.shortcodes) == 0 {
			t.Fatalf("theme %v is incomplete", candidate.keywords)
		}
		for _, shortcode := range candidate.shortcodes {
			if !catalog.Allows(shortcode) {
				t.Errorf("theme %q lists off-catalog %q", candidate.keywords[0], shortcode)
			}
			if !catalog.inPool[shortcode] {
				t.Errorf("theme %q lists non-suggestible %q", candidate.keywords[0], shortcode)
			}
			if IsProhibited(shortcode) || IsNonRendering(shortcode) {
				t.Errorf("theme %q lists unusable %q", candidate.keywords[0], shortcode)
			}
		}
	}
}

func TestThemeMatches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		subject string
		want    string
	}{
		{"lint cleanup", "Clear the standing linter findings", "necktie"},
		{"test work", "Add tests for the decoder", "tea"},
		{"docs", "Document the commit convention", "memo"},
		{"persistence", "Add the repository cache", "floppy_disk"},
		{"authz", "Verify the JWT at the middleware", "closed_lock_with_key"},
		{"ai", "Queue the agent auto sends", "honeybee"},
		{"perf", "Optimize the hot path", "zap"},
		{"container", "Pin the docker image by digest", "whale"},
		{"git hook", "Pass the project dir to the commit hook", "fishing_pole_and_fish"},
		{"routing", "Add the REST router contract", "twisted_rightwards_arrows"},
		{"no theme", "Adjust the widget count", ""},
	}

	catalog := mustCatalog(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			matched := catalog.themeMatches(testCase.subject)
			if testCase.want == "" {
				if len(matched) != 0 {
					t.Fatalf("expected no theme, got %v", matched)
				}
				return
			}
			if !slices.Contains(matched, testCase.want) {
				t.Fatalf("themeMatches(%q) = %v, want it to contain %q",
					testCase.subject, matched, testCase.want)
			}
		})
	}
}

func TestThemeMatchesIsCaseAndSuffixTolerant(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"Add TESTS for the parser",
		"Refactoring the dispatcher",
		"Fixes a crash",
	}
	catalog := mustCatalog(t)
	for _, subject := range testCases {
		t.Run(subject, func(t *testing.T) {
			t.Parallel()
			if len(catalog.themeMatches(subject)) == 0 {
				t.Fatalf("themeMatches(%q) found nothing", subject)
			}
		})
	}
}

func loadAnchorFixture(t *testing.T) ([]Anchor, []theme) {
	t.Helper()

	catalog := mustCatalog(t)
	return catalog.Anchors(), catalog.themes
}

func TestAnchors_MirrorsTheLoadedThemeTableInOrder(t *testing.T) {
	t.Parallel()

	anchors, themes := loadAnchorFixture(t)
	if len(anchors) != len(themes) {
		t.Fatalf("Anchors() returned %d entries, want %d", len(anchors), len(themes))
	}
	for idx, want := range themes {
		if anchors[idx].Area != want.keywords[0] {
			t.Errorf("anchor %d area = %q, want %q", idx, anchors[idx].Area, want.keywords[0])
		}
		if !slices.Equal(anchors[idx].Shortcodes, want.shortcodes) {
			t.Errorf(
				"anchor %d shortcodes = %v, want %v",
				idx,
				anchors[idx].Shortcodes,
				want.shortcodes,
			)
		}
	}
}

func TestAnchors_MoveArea(t *testing.T) {
	t.Parallel()

	anchors, _ := loadAnchorFixture(t)
	var move Anchor
	found := false
	for _, anchor := range anchors {
		if anchor.Area == "move" {
			move, found = anchor, true
			break
		}
	}
	if !found {
		t.Fatal("expected a move anchor")
	}
	if !slices.Contains(move.Shortcodes, "open_file_folder") {
		t.Errorf("move anchor = %v, want it to contain open_file_folder", move.Shortcodes)
	}
}

func TestSuggestLeadsWithTheMatchingTheme(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		commitType string
		subject    string
		wantOneOf  []string
	}{
		{
			"lint chore",
			"chore",
			"Clear the standing linter findings",
			[]string{"necktie", "art", "bathtub"},
		},
		{
			"test work",
			"test",
			"Cover the decoder with tests",
			[]string{"tea", "microscope", "hourglass", "white_check_mark", "telescope"},
		},
		{
			"persistence",
			"feat",
			"Add the repository cache",
			[]string{
				"floppy_disk",
				"file_folder",
				"card_file_box",
				"open_file_folder",
				"minidisc",
			},
		},
		{
			"docs",
			"docs",
			"Document the commit convention",
			[]string{"memo", "page_with_curl", "books", "green_book", "pencil2", "book"},
		},
	}

	catalog := mustCatalog(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			picks := catalog.Suggest(testCase.commitType, testCase.subject, nil, 7)
			if len(picks) == 0 {
				t.Fatal("Suggest returned nothing")
			}
			if !slices.Contains(testCase.wantOneOf, picks[0]) {
				t.Fatalf("first pick %q is off-theme; suggestions were %v", picks[0], picks)
			}
		})
	}
}

func TestSuggest_WidensBeyondTheme(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	picks := catalog.Suggest("test", "Cover the decoder with tests", nil, 7)
	family := catalog.themeMatches("Cover the decoder with tests")

	offTheme := 0
	for _, pick := range picks {
		if !slices.Contains(family, pick) {
			offTheme++
		}
	}
	if offTheme == 0 {
		t.Fatalf("every suggestion came from one family: %v", picks)
	}
}
