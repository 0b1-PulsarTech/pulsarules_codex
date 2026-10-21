package governance

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

type Result struct {
	Findings            []finding.Finding
	SuppressedGenerated int
}
