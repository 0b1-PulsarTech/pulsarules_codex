package main

// emojigen regenerates the embedded commit-emoji catalog. Manual tool, not
// part of the build: it reaches the network and a local clone of the
// reference repository. Run only when the catalog changes.
//
//	go run ./tools/emojigen --jane <path-to-jane> --out internal/emoji/data

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

func main() {
	janeDir := flag.String("jane", "", "path to the jane repository clone")
	outDir := flag.String("out", "internal/emoji/data", "data output directory")
	flag.Parse()

	if *janeDir == "" {
		_, _ = fmt.Fprintln(os.Stderr, "emojigen: --jane is required")
		os.Exit(2)
	}

	if err := run(*janeDir, *outDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "emojigen: %v\n", err)
		os.Exit(1)
	}
}

func run(janeDir, outDir string) error {
	shortcodes, err := fetchShortcodes()
	if err != nil {
		return err
	}
	meta, err := fetchUnicodeMeta()
	if err != nil {
		return err
	}
	usage, err := readJaneUsage(janeDir)
	if err != nil {
		return err
	}

	entries := buildCatalog(shortcodes, meta, usage.byEmoji)
	if mkdirErr := os.MkdirAll(outDir, fsperm.DirPrivate); mkdirErr != nil {
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
