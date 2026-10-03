package analysis

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
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

func FormatFindings(findings []core.Finding, style FindingStyle, header string) string {
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
		escape.Line(finding.AnalyzerID),
		escape.Line(finding.Message),
	)
	if loc := location(finding); loc != "" {
		out.WriteString(" (" + loc + ")")
	}
	out.WriteByte('\n')

	if finding.Suggestion != "" {
		out.WriteString(form.arrow + "\u2192 " + escape.Line(finding.Suggestion) + "\n")
	}
	if form.withRule {
		// The full summary is a ~400 char paragraph that drowns the finding.
		if summary := strings.TrimSpace(finding.RuleSummary); summary != "" {
			rule := escape.Line(knowledge.FirstSentence(summary))
			out.WriteString(form.arrow + "rule: " + rule + "\n")
		}
	}
}

func location(finding core.Finding) string {
	if finding.File == "" {
		return ""
	}
	if finding.Line > 0 {
		return fmt.Sprintf("%s:%d", escape.Path(finding.File), finding.Line)
	}
	return escape.Path(finding.File)
}

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
