package emoji

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io/fs"
)

//go:embed data/catalog.txt
var dataFS embed.FS

type Catalog struct {
	allowed map[string]struct{}
}

func NewCatalog() (*Catalog, error) {
	const estimatedCatalogSize = 1100
	catalog := &Catalog{allowed: make(map[string]struct{}, estimatedCatalogSize)}
	if err := catalog.readCatalog(dataFS); err != nil {
		return nil, err
	}
	return catalog, nil
}

func (catalog *Catalog) readCatalog(fsys fs.FS) error {
	const catalogFieldCount = 4
	// The catalog.txt columns are name, group, subgroup and jane_count, the last being
	// the measured usage count.
	records, err := readRecords(fsys, "data/catalog.txt", catalogFieldCount)
	if err != nil {
		return err
	}
	for _, fields := range records {
		catalog.allowed[fields[0]] = struct{}{}
	}
	return nil
}

func (catalog *Catalog) Allows(shortcode string) bool {
	_, found := catalog.allowed[shortcode]
	return found
}

func (catalog *Catalog) Size() int { return len(catalog.allowed) }

func readRecords(fsys fs.FS, name string, fieldCount int) ([][]string, error) {
	// Takes fsys so tests can pass a fabricated fs.FS.
	file, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.Comment = '#'
	reader.FieldsPerRecord = fieldCount

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return records, nil
}
