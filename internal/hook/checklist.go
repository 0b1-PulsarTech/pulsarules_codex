package hook

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

// checklistByExt maps a changed file's extension to the reminders worth raising
// before a commit. The Stop hook unions the bullets for the extensions present
// in the working tree. This is the default set; it will be overridden via
// config in a later phase.
var checklistByExt = map[string][]string{
	".go": {
		"gofmt -l clean; go vet ./... and go test -race ./... green",
		"no package docstrings; no doc comments on unexported symbols",
	},
	".md":   {"every Markdown file ends with a trailing newline"},
	".sql":  {"migrations reversible and reviewed; no raw SQL in handlers"},
	".yaml": {"validate passes: go run ./cmd/pulsarules_cli validate"},
}

// extOrder fixes the bullet order so identical change sets produce identical
// output (the dedup in SessionTracker.FirstEmission is content-based).
var extOrder = []string{".go", ".sql", ".yaml", ".md"}

// TypedChecklist builds the per-file-type reminder bullets for the
// extensions present in status, in a stable order with no duplicate bullets.
func TypedChecklist(status vcs.Status) string {
	present := status.Extensions()
	var bullets []string
	seen := map[string]bool{}
	for _, ext := range extOrder {
		if !present[ext] {
			continue
		}
		for _, bullet := range checklistByExt[ext] {
			if !seen[bullet] {
				seen[bullet] = true
				bullets = append(bullets, "  - "+bullet)
			}
		}
	}
	return strings.Join(bullets, "\n")
}
