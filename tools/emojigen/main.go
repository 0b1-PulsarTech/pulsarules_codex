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
	if mkdirErr := os.MkdirAll(outDir, disk.DirPrivate); mkdirErr != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, mkdirErr)
	}
	if writeCatalogErr := writeCatalog(
		filepath.Join(outDir, "catalog.txt"),
		entries,
	); writeCatalogErr != nil {
		return writeCatalogErr
	}

	fmt.Printf("emojigen: %d catalog entries\n", len(entries))
	return nil
}
