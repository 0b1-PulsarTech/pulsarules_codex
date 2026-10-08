package command

import (
	"encoding/json"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type dartReport struct {
	Version     int              `json:"version"`
	Diagnostics []dartDiagnostic `json:"diagnostics"`
}

type dartDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Location struct {
		File  string `json:"file"`
		Range struct {
			Start struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"range"`
	} `json:"location"`
	ProblemMessage    string `json:"problemMessage"`
	CorrectionMessage string `json:"correctionMessage"`
}

func decodeDartAnalyze(output []byte) ([]diagnostic, error) {
	const dartReportVersion = 1

	var report dartReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, fmt.Errorf("decode dart analyze json: %w", err)
	}
	if report.Version != dartReportVersion {
		return nil, fmt.Errorf(
			"dart analyze json version %d, want %d", report.Version, dartReportVersion,
		)
	}
	found := make([]diagnostic, 0, len(report.Diagnostics))
	for _, item := range report.Diagnostics {
		found = append(found, diagnostic{
			code:       item.Code,
			severity:   dartSeverity(item.Severity),
			file:       item.Location.File,
			line:       item.Location.Range.Start.Line,
			message:    item.ProblemMessage,
			suggestion: item.CorrectionMessage,
		})
	}
	return found, nil
}

func dartSeverity(name string) finding.Severity {
	switch name {
	case "ERROR":
		return finding.SeverityError
	case "INFO":
		return finding.SeverityInfo
	default:
		// An unknown level is surfaced, not hidden: warning is the middle rung.
		return finding.SeverityWarning
	}
}
