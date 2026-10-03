package checklist

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestTyped(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		changes []vcs.Change
		expect  string
	}{
		{name: "empty status produces empty checklist"},
		{
			name: "go and sql in fixed order",
			changes: []vcs.Change{
				{Path: "schema.sql", Extension: ".sql"},
				{Path: "main.go", Extension: ".go"},
			},
			expect: "  - gofmt -l clean; go vet ./... and go test -race ./... green\n" +
				"  - no package docstrings; no doc comments on unexported symbols\n" +
				"  - migrations reversible and reviewed; no raw SQL in handlers",
		},
		{
			name: "two files of one extension list its bullets once",
			changes: []vcs.Change{
				{Path: "a.md", Extension: ".md"},
				{Path: "b.md", Extension: ".md"},
			},
			expect: "  - every Markdown file ends with a trailing newline",
		},
		{
			name: "yaml then md",
			changes: []vcs.Change{
				{Path: "README.md", Extension: ".md"},
				{Path: "skills.yaml", Extension: ".yaml"},
			},
			expect: "  - validate passes: go run ./cmd/pulsarules_cli validate\n" +
				"  - every Markdown file ends with a trailing newline",
		},
		{
			name:    "unknown extension has no bullets",
			changes: []vcs.Change{{Path: "main.rs", Extension: ".rs"}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := Typed(vcs.Status{Changes: testCase.changes})
			if got != testCase.expect {
				t.Fatalf("Typed() = %q, want %q", got, testCase.expect)
			}
		})
	}
}
