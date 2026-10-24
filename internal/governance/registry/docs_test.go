package registry

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAnalyzerDocs_MatchRegistry(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "analyzers.md"))
	if err != nil {
		t.Fatalf("read docs/analyzers.md: %v", err)
	}
	scopes := map[Gate]string{GateStatic: "commit", GateArch: "changed", GateFullOnly: "full"}
	rows, params := map[string]string{}, map[string][]string{}
	for line := range strings.Lines(string(raw)) {
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 3 || !strings.HasPrefix(strings.TrimSpace(cells[0]), "`") {
			continue
		}
		id, second := strings.Trim(strings.TrimSpace(cells[0]), "`"), strings.TrimSpace(cells[1])
		if strings.HasPrefix(second, "`") {
			params[id] = append(params[id], strings.Trim(second, "`"))
			continue
		}
		rows[id] = second + " " + strings.Trim(strings.TrimSpace(cells[2]), "`")
	}
	testCases := slices.DeleteFunc(specs(), func(spec entry) bool { return spec.internal })
	for _, testCase := range testCases {
		t.Run(testCase.ID, func(t *testing.T) {
			t.Parallel()

			want := scopes[testCase.gate] + " " + cmp.Or(testCase.Rule, "-")
			if got := rows[testCase.ID]; got != want {
				t.Errorf("scope and rule = %q, want %q", got, want)
			}
			if got := params[testCase.ID]; !slices.Equal(got, testCase.Params) {
				t.Errorf("params = %v, want %v", got, testCase.Params)
			}
		})
	}
	for id := range params {
		if _, listed := rows[id]; !listed {
			t.Errorf("params listed for %q, which the analyzer table lacks", id)
		}
	}
	if len(rows) != len(testCases) {
		t.Errorf("docs list %d analyzers, the registry registers %d", len(rows), len(testCases))
	}
}
