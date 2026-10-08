package markdown

import (
	"slices"
	"strings"
	"testing"
)

func TestReading_Parse(t *testing.T) {
	t.Parallel()

	quoted := reading{quotes: true}
	tabbed := reading{tabs: true}
	testCases := []struct {
		name string
		how  reading
		text string
		want lineShape
	}{
		{"prose", reading{}, "text\n", lineShape{}},
		{"blank", reading{}, "  \n", lineShape{indent: 2, blank: true}},
		{"bare fence", reading{}, "```\n", lineShape{char: '`', run: 3, bare: true}},
		{"info string", reading{}, "~~~go\n", lineShape{char: '~', run: 3}},
		{
			"tick in the info string",
			reading{},
			"```a`\n",
			lineShape{char: '`', run: 3, tickInfo: true},
		},
		{"quotes are content when not read", reading{}, "> ```\n", lineShape{}},
		{"one quote", quoted, "> ```\n", lineShape{depth: 1, char: '`', run: 3, bare: true}},
		{"nested quotes", quoted, "> > ```\n", lineShape{depth: 2, char: '`', run: 3, bare: true}},
		{"quote-only line", quoted, ">\n", lineShape{depth: 1, blank: true}},
		{"a quote mark after four spaces is text", quoted, "    > ```\n", lineShape{
			indent: 4, indented: true,
		}},
		{"indent counts spaces", reading{}, "  ```\n", lineShape{
			indent: 2, char: '`', run: 3, bare: true,
		}},
		{"a tab is four columns", tabbed, "\t```\n", lineShape{
			indent: 4, indented: true, char: '`', run: 3, bare: true,
		}},
		{"a tab ends a space-only indent", reading{}, "\t```\n", lineShape{indented: true}},
		{"four spaces are indented", reading{}, "    x\n", lineShape{indent: 4, indented: true}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.how.parse(testCase.text); got != testCase.want {
				t.Errorf("parse(%q) = %+v, want %+v", testCase.text, got, testCase.want)
			}
		})
	}
}

func regionsOf(how reading, src string) []Region {
	scan := &readingScan{how: how}
	offset := 0
	for _, text := range strings.SplitAfter(src, "\n") {
		scan.line(text, offset)
		offset += len(text)
	}
	return scan.finish(len(src))
}

func TestReadings_KeepThePreviousDetectors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		index int
		src   string
		want  []Region
	}{
		{"zero reads no quotes", 0, "> ```\nx\n", nil},
		{"zero opens on a tick info", 0, "```a`\nx\n", []Region{{0, 8}}},
		{"zero closes on text after the closer", 0, "```\nx\n```y\nz\n", []Region{{0, 11}}},
		{"zero reads no indented code", 0, "p\n\n    x\n", nil},
		{"one reads a quoted fence", 1, "> ```\n> x\n", []Region{{0, 10}}},
		{"one reads a fence shifted three spaces", 1, "   > ```\n> x\n", []Region{{0, 13}}},
		{"one reads no fence shifted four spaces", 1, "    > ```\n> x\n", nil},
		{"one skips a tick info", 1, "```a`\nx\n", nil},
		{"one wants a bare closer", 1, "```\nx\n```y\nz\n", []Region{{0, 13}}},
		{"one reads indented code", 1, "p\n\n    x\n", []Region{{3, 9}}},
		{"one reads no tab after a space", 1, " \t```\nx\n```\n", []Region{{8, 12}}},
		{"one closes without the strict rules", 1, "```\n    ```\nx\n", []Region{{0, 12}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := regionsOf(readings()[testCase.index], testCase.src)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("regions of %q = %v, want %v", testCase.src, got, testCase.want)
			}
		})
	}
}
