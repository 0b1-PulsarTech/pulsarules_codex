package findings

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type FindingStyle string

const (
	StyleCLI  FindingStyle = "cli"
	StyleHook FindingStyle = "hook"
)

type layout struct {
	indent   string
	arrow    string
	upper    bool
	withRule bool
}

func layoutFor(style FindingStyle) layout {
	if style == StyleHook {
		return layout{indent: "  ", arrow: "    ", upper: false, withRule: false}
	}
	return layout{indent: "", arrow: "  ", upper: true, withRule: true}
}

func FormatFindings(findings []pass.Finding, style FindingStyle, header string) string {
	if len(findings) == 0 {
		return ""
	}
	form := layoutFor(style)

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

func writeFinding(out *strings.Builder, finding pass.Finding, form layout) {
	label := severityLabel(finding.Severity)
	if form.upper {
		label = strings.ToUpper(label)
	}
	_, _ = fmt.Fprintf(
		out,
		"%s[%s] %s: %s",
		form.indent,
		label,
		printable.Line(finding.AnalyzerID),
		printable.Line(finding.Message),
	)
	if loc := location(finding); loc != "" {
		out.WriteString(" (" + loc + ")")
	}
	out.WriteByte('\n')

	if finding.Suggestion != "" {
		out.WriteString(form.arrow + "\u2192 " + printable.Line(finding.Suggestion) + "\n")
	}
	if form.withRule {
		// The full summary is a ~400 char paragraph that drowns the finding.
		if summary := strings.TrimSpace(finding.RuleSummary); summary != "" {
			rule := printable.Line(knowledge.FirstSentence(summary))
			out.WriteString(form.arrow + "rule: " + rule + "\n")
		}
	}
}

func location(finding pass.Finding) string {
	if finding.File == "" {
		return ""
	}
	if finding.Line > 0 {
		return fmt.Sprintf("%s:%d", printable.Path(finding.File), finding.Line)
	}
	return printable.Path(finding.File)
}

func severityLabel(severity pass.Severity) string {
	switch severity {
	case pass.SeverityWarning:
		return "warning"
	case pass.SeverityError:
		return "error"
	default:
		return "info"
	}
}
