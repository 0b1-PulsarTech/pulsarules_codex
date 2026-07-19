package emoji

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

//go:embed data/catalog.txt data/types.txt data/themes.txt
var dataFS embed.FS

type Catalog struct {
	allowed map[string]entry
	pool    []string
	inPool  map[string]bool
	byType  map[string][]string
	themes  []theme
}

type entry struct {
	group    string
	subgroup string
	useCount int
}

func NewCatalog() (*Catalog, error) {
	const (
		estimatedCatalogSize     = 1100
		estimatedCommitTypeCount = 16
	)
	catalog := &Catalog{
		allowed: make(map[string]entry, estimatedCatalogSize),
		byType:  make(map[string][]string, estimatedCommitTypeCount),
	}
	if err := catalog.readCatalog(dataFS); err != nil {
		return nil, err
	}
	if err := catalog.readTypes(); err != nil {
		return nil, err
	}
	themes, err := loadThemes()
	if err != nil {
		return nil, fmt.Errorf("load themes: %w", err)
	}
	catalog.themes = themes
	catalog.buildPool()
	return catalog, nil
}

func (catalog *Catalog) readCatalog(fsys fs.FS) error {
	const (
		catalogFieldCount  = 4
		useCountFieldIndex = 3
	)
	// The catalog.txt columns are name, group, subgroup and use_count, the last being
	// the measured usage count.
	records, err := readRecords(fsys, "data/catalog.txt", catalogFieldCount)
	if err != nil {
		return err
	}
	for _, fields := range records {
		count, countErr := strconv.Atoi(fields[useCountFieldIndex])
		if countErr != nil {
			continue
		}
		catalog.allowed[fields[0]] = entry{group: fields[1], subgroup: fields[2], useCount: count}
	}
	return nil
}

func (catalog *Catalog) readTypes() error {
	const typesFieldCount = 3
	// The types.txt columns are type, name and count.
	records, err := readRecords(dataFS, "data/types.txt", typesFieldCount)
	if err != nil {
		return err
	}
	for _, fields := range records {
		commitType, shortcode := fields[0], fields[1]
		if _, ok := catalog.allowed[shortcode]; !ok {
			continue
		}
		catalog.byType[commitType] = append(catalog.byType[commitType], shortcode)
	}
	return nil
}

func (catalog *Catalog) buildPool() {
	catalog.pool = make([]string, 0, len(catalog.allowed))
	for shortcode, info := range catalog.allowed {
		// Measured usage outranks the group lists: an emoji real commits
		// already reached for is proven to read as an engineering area, wherever
		// Unicode happens to file it.
		if info.useCount > 0 ||
			suggestibleGroup(info.group) ||
			suggestibleSubgroup(info.subgroup) {
			catalog.pool = append(catalog.pool, shortcode)
		}
	}
	slices.SortFunc(catalog.pool, func(left, right string) int {
		if diff := catalog.allowed[right].useCount - catalog.allowed[left].useCount; diff != 0 {
			return diff
		}
		return strings.Compare(left, right)
	})

	catalog.inPool = make(map[string]bool, len(catalog.pool))
	for _, shortcode := range catalog.pool {
		catalog.inPool[shortcode] = true
	}
}

func (catalog *Catalog) Allows(shortcode string) bool {
	_, found := catalog.allowed[shortcode]
	return found
}

func (catalog *Catalog) Size() int { return len(catalog.allowed) }

func (catalog *Catalog) PoolSize() int { return len(catalog.pool) }

func suggestibleGroup(group string) bool {
	switch group {
	// The groups the reference repository draws commit emoji from by measured usage.
	// Membership decides what may be suggested, never what is allowed.
	case "Objects", "Travel & Places", "Animals & Nature", "Food & Drink", "Activities":
		return true
	}
	return false
}

func suggestibleSubgroup(subgroup string) bool {
	switch subgroup {
	// Corners of noisy groups that read as "hide/quiet this" better than Objects.
	case "other-symbol", "warning", "punctuation", "av-symbol", "flag", "monkey-face":
		return true
	}
	return false
}

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
