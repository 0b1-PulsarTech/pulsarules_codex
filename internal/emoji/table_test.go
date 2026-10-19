package emoji

import (
	"slices"
	"testing"
	"testing/fstest"
)

func TestFormatTable_RoundTripsThroughReadRecords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		header     string
		rows       [][]string
		fieldCount int
		wantText   string
	}{
		{
			name:       "catalog rows",
			header:     CatalogHeader,
			rows:       [][]string{CatalogRow("wrench", "Objects", "tool", 3)},
			fieldCount: 4,
			wantText:   CatalogHeader + "wrench\tObjects\ttool\t3\n",
		},
		{
			name:       "types rows",
			header:     TypesHeader,
			rows:       [][]string{TypesRow("build", "whale", 17)},
			fieldCount: 3,
			wantText:   TypesHeader + "build\twhale\t17\n",
		},
		{name: "header only", header: TypesHeader, wantText: TypesHeader, fieldCount: 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			text := FormatTable(testCase.header, testCase.rows)
			if text != testCase.wantText {
				t.Fatalf("FormatTable = %q, want %q", text, testCase.wantText)
			}
			fsys := fstest.MapFS{"table.txt": {Data: []byte(text)}}
			got, err := readRecords(fsys, "table.txt", testCase.fieldCount)
			if err != nil {
				t.Fatalf("readRecords: %v", err)
			}
			if !slices.EqualFunc(got, testCase.rows, slices.Equal) {
				t.Errorf("read back %v, want %v", got, testCase.rows)
			}
		})
	}
}
