package commit

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"

func hasAnalyzer(findings []core.Finding, id string) bool {
	_, ok := findingByID(findings, id)
	return ok
}

func findingByID(findings []core.Finding, id string) (core.Finding, bool) {
	for _, finding := range findings {
		if finding.AnalyzerID == id {
			return finding, true
		}
	}
	return core.Finding{}, false
}
