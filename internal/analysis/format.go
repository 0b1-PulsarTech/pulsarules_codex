package analysis

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// FindingStyle selects the layout FormatFindings renders.
type FindingStyle string

const (
	// StyleCLI is the flush-left, uppercase-severity layout the CLI commands
	// print for a human or a git hook to read on stderr.
	StyleCLI FindingStyle = "cli"
	// StyleHook is the indented, lowercase-severity layout embedded in the
	// additionalContext a Claude Code hook emits.
	StyleHook FindingStyle = "hook"
)

// layout holds the per-style spacing and casing so the renderer stays one loop
// instead of a switch repeated per line.
type layout struct {
	indent   string
	arrow    string
	upper    bool
	withRule bool
}

var layouts = map[FindingStyle]layout{
	StyleCLI:  {indent: "", arrow: "  ", upper: true, withRule: true},
	StyleHook: {indent: "  ", arrow: "    ", upper: false, withRule: false},
}

// FormatFindings renders findings as a text block in the given style, prefixed
// by header when that is non-empty. It returns the empty string for an empty
// finding set, so a caller can test the rendered result rather than the slice.
func FormatFindings(findings []core.Finding, style FindingStyle, header string) string {
	if len(findings) == 0 {
		return ""
	}
	form, ok := layouts[style]
	if !ok {
		form = layouts[StyleCLI]
	}

	var out strings.Builder
	if header != "" {
		out.WriteString(header)
		out.WriteByte('\n')
	}
	for _, finding := range findings {
		writeFinding(&out, finding, form)
	}
	return out.String()
}

func writeFinding(out *strings.Builder, finding core.Finding, form layout) {
	label := severityLabel(finding.Severity)
	if form.upper {
		label = strings.ToUpper(label)
	}
	_, _ = fmt.Fprintf(
		out,
		"%s[%s] %s: %s",
		form.indent,
		label,
		finding.AnalyzerID,
		finding.Message,
	)
	if loc := location(finding); loc != "" {
		out.WriteString(" (" + loc + ")")
	}
	out.WriteByte('\n')

	if finding.Suggestion != "" {
		out.WriteString(form.arrow + "→ " + finding.Suggestion + "\n")
	}
	if form.withRule {
		// why: a rule's blockquote summary is a whole paragraph joined into one
		// line - up to ~400 chars - which drowns the finding it annotates.
		if summary := strings.TrimSpace(finding.RuleSummary); summary != "" {
			out.WriteString(form.arrow + "rule: " + knowledge.FirstSentence(summary) + "\n")
		}
	}
}

func location(finding core.Finding) string {
	if finding.File == "" {
		return ""
	}
	if finding.Line > 0 {
		return fmt.Sprintf("%s:%d", finding.File, finding.Line)
	}
	return finding.File
}

// severityLabel keeps the vocabulary core.ParamSet.Severity parses back
// (core.ParamSeverity), so a label always round-trips as a param. Casing is
// the caller's choice (writeFinding), not a flag argument here.
func severityLabel(severity core.Severity) string {
	switch severity {
	case core.SeverityWarning:
		return "warning"
	case core.SeverityError:
		return "error"
	default:
		return "info"
	}
}
