package emoji

import (
	"slices"
	"testing"
)

func TestThemeShortcodesAreUsable(t *testing.T) {
	t.Parallel()

	catalog := mustCatalog(t)
	themes, err := loadThemes()
	if err != nil {
		t.Fatalf("loadThemes: %v", err)
	}
	for _, candidate := range themes {
		if len(candidate.keywords) == 0 || len(candidate.shortcodes) == 0 {
			t.Fatalf("theme %v is incomplete", candidate.keywords)
		}
		for _, shortcode := range candidate.shortcodes {
			if !catalog.Allows(shortcode) {
				t.Errorf("theme %q lists off-catalog %q", candidate.keywords[0], shortcode)
			}
			if IsProhibited(shortcode) || IsNonRendering(shortcode) {
				t.Errorf("theme %q lists unusable %q", candidate.keywords[0], shortcode)
			}
		}
	}
}

func loadAnchorFixture(t *testing.T) ([]Anchor, []theme) {
	t.Helper()

	anchors, err := Anchors()
	if err != nil {
		t.Fatalf("Anchors: %v", err)
	}
	themes, err := loadThemes()
	if err != nil {
		t.Fatalf("loadThemes: %v", err)
	}
	return anchors, themes
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
