package emoji

import (
	"encoding/csv"
	"fmt"
	"io/fs"
)

// readRecords takes fsys rather than always reading the package-level dataFS
// so tests can point it at a fabricated fs.FS instead of corrupting the real
// embedded data.
func readRecords(fsys fs.FS, name string, fieldCount int) ([][]string, error) {
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
