package emoji

import (
	"encoding/csv"
	"fmt"
	"io/fs"
)

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
