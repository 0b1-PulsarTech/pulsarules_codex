package message

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

func hasAnalyzer(findings []pass.Finding, id string) bool {
	_, ok := findingByID(findings, id)
	return ok
}

func findingByID(findings []pass.Finding, id string) (pass.Finding, bool) {
	for _, finding := range findings {
		if finding.AnalyzerID == id {
			return finding, true
		}
	}
	return pass.Finding{}, false
}
