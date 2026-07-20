package message

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

func hasAnalyzer(findings []finding.Finding, id string) bool {
	_, ok := findingByID(findings, id)
	return ok
}

func findingByID(findings []finding.Finding, id string) (finding.Finding, bool) {
	for _, finding := range findings {
		if finding.AnalyzerID == id {
			return finding, true
		}
	}
	return finding.Finding{}, false
}
