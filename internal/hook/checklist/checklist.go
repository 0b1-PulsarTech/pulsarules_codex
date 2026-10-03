package checklist

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func Typed(status vcs.Status) string {
	present := status.Extensions()
	var bullets []string
	// A fixed order keeps output stable for the content-based dedup in SessionTracker.
	for _, ext := range []string{".go", ".sql", ".yaml", ".md"} {
		if !present[ext] {
			continue
		}
		for _, bullet := range bulletsFor(ext) {
			bullets = append(bullets, "  - "+bullet)
		}
	}
	return strings.Join(bullets, "\n")
}

func bulletsFor(ext string) []string {
	switch ext {
	case ".go":
		return []string{
			"gofmt -l clean; go vet ./... and go test -race ./... green",
			"no package docstrings; no doc comments on unexported symbols",
		}
	case ".md":
		return []string{"every Markdown file ends with a trailing newline"}
	case ".sql":
		return []string{"migrations reversible and reviewed; no raw SQL in handlers"}
	case ".yaml":
		return []string{"validate passes: go run ./cmd/pulsarules_cli validate"}
	}
	return nil
}
