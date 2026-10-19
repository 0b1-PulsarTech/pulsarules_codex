package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

type catalogEntry struct {
	Name     string
	Version  float64
	Group    string
	Subgroup string
	UseCount int
}

func buildCatalog(
	shortcodes map[string]string,
	meta map[string]unicodeMeta,
	useCounts map[string]int,
) []catalogEntry {
	entries := make([]catalogEntry, 0, len(shortcodes))
	for name, codepoints := range shortcodes {
		sequenceMeta, ok := lookupMeta(meta, codepoints)
		if !ok {
			continue
		}
		if !admits(name, sequenceMeta) {
			continue
		}
		entries = append(entries, catalogEntry{
			Name:     name,
			Version:  sequenceMeta.Version,
			Group:    sequenceMeta.Group,
			Subgroup: sequenceMeta.Subgroup,
			UseCount: useCounts[name],
		})
	}
	slices.SortFunc(entries, func(left, right catalogEntry) int {
		return strings.Compare(left.Name, right.Name)
	})
	return entries
}

func admits(name string, sequenceMeta unicodeMeta) bool {
	const maxEmojiVersion = 1.0

	if emoji.IsProhibited(name) || emoji.IsNonRendering(name) {
		return false
	}
	// Country flags carry no engineering meaning and would crowd the catalog;
	// non-country flags (checkered_flag, crossed_flags) stay.
	switch sequenceMeta.Subgroup {
	case "country-flag", "subdivision-flag":
		return false
	}
	// GitKraken renders up to this version: every shortcode confirmed broken by hand
	// sits above it, and 286 of 295 in the reference repository sit at or below it.
	if sequenceMeta.Version <= maxEmojiVersion {
		return true
	}
	// Above the cutoff, but the reference repository uses them, so they are kept; the
	// first to re-check if raw text shows up in a commit list again.
	switch name {
	case "beetle", "egg", "safety_vest", "shopping_cart", "pirate_flag":
		return true
	}
	return false
}

func lookupMeta(meta map[string]unicodeMeta, codepoints string) (unicodeMeta, bool) {
	for _, candidate := range []string{
		codepoints,
		stripJoiners(codepoints),
		codepoints + "-fe0f",
	} {
		if sequenceMeta, ok := meta[candidate]; ok {
			return sequenceMeta, true
		}
	}
	return unicodeMeta{}, false
}

func writeCatalog(path string, entries []catalogEntry) error {
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(
			rows,
			emoji.CatalogRow(entry.Name, entry.Group, entry.Subgroup, entry.UseCount),
		)
	}
	return writeTable(path, emoji.CatalogHeader, rows)
}

func writeTypes(path string, byType map[string]map[string]int, allowed map[string]bool) error {
	var rows [][]string
	for _, commitType := range slices.Sorted(maps.Keys(byType)) {
		for _, entry := range rankedCounts(byType[commitType], allowed) {
			rows = append(rows, emoji.TypesRow(commitType, entry.name, entry.count))
		}
	}
	return writeTable(path, emoji.TypesHeader, rows)
}

func writeTable(path, header string, rows [][]string) error {
	if err := os.WriteFile(
		path,
		[]byte(emoji.FormatTable(header, rows)),
		disk.FilePrivate,
	); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

type rankedEntry struct {
	name  string
	count int
}

func rankedCounts(counts map[string]int, allowed map[string]bool) []rankedEntry {
	ranked := make([]rankedEntry, 0, len(counts))
	for name, count := range counts {
		if allowed[name] {
			ranked = append(ranked, rankedEntry{name: name, count: count})
		}
	}
	// Ties break by name so the generated file is stable across runs.
	slices.SortFunc(ranked, func(left, right rankedEntry) int {
		if left.count != right.count {
			return right.count - left.count
		}
		return strings.Compare(left.name, right.name)
	})
	return ranked
}
