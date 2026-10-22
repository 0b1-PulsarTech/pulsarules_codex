package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "emojigen: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	const outDir = "internal/emoji/data"

	shortcodes, err := fetchShortcodes()
	if err != nil {
		return err
	}
	meta, err := fetchUnicodeMeta()
	if err != nil {
		return err
	}
	usage, err := readUsage(".")
	if err != nil {
		return err
	}

	entries := buildCatalog(shortcodes, meta, usage.byEmoji)
	if mkdirErr := os.MkdirAll(outDir, disk.ModeDirPrivate); mkdirErr != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, mkdirErr)
	}
	if writeCatalogErr := writeCatalog(
		filepath.Join(outDir, "catalog.txt"),
		entries,
	); writeCatalogErr != nil {
		return writeCatalogErr
	}
	allowed := make(map[string]bool, len(entries))
	for _, entry := range entries {
		allowed[entry.Name] = true
	}
	if writeTypesErr := writeTypes(
		filepath.Join(outDir, "types.txt"),
		usage.byType,
		allowed,
	); writeTypesErr != nil {
		return writeTypesErr
	}

	fmt.Printf("emojigen: %d catalog entries, %d typed subjects\n", len(entries), usage.typed)
	return nil
}
