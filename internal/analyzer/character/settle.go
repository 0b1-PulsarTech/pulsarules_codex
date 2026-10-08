package character

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

type hintedFinding struct {
	finding finding.Finding
	caution int
}

type lineMessage struct {
	line    int
	message string
}

func settle(items []hintedFinding) []finding.Finding {
	strongest := make(map[lineMessage]hintedFinding, len(items))
	// Output keeps one finding per file, line and message, so the first hint would
	// stand for the line: give every finding the most cautious hint instead.
	for _, item := range items {
		key := lineMessage{line: item.finding.Line, message: item.finding.Message}
		if seen, found := strongest[key]; !found || item.caution > seen.caution {
			strongest[key] = item
		}
	}
	findings := make([]finding.Finding, 0, len(items))
	for _, item := range items {
		key := lineMessage{line: item.finding.Line, message: item.finding.Message}
		item.finding.Suggestion = strongest[key].finding.Suggestion
		findings = append(findings, item.finding)
	}
	return findings
}
