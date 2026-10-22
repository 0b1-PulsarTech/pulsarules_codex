package findings

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Layout struct {
	indent   string
	arrow    string
	upper    bool
	withRule bool
}

func CLI() Layout {
	return Layout{indent: "", arrow: "  ", upper: true, withRule: true}
}

func Hook() Layout {
	return Layout{indent: "  ", arrow: "    ", upper: false, withRule: false}
}

func (report Report) Text(form Layout, header string) string {
	if len(report.Findings) == 0 {
		return ""
	}

	var out strings.Builder
	if header != "" {
		out.WriteString(header)
		out.WriteByte('\n')
	}
	for _, hit := range report.Findings {
		writeFinding(&out, hit, form)
	}
	return out.String()
}

func writeFinding(out *strings.Builder, hit finding.Finding, form Layout) {
	label := hit.Severity.String()
	if form.upper {
		label = strings.ToUpper(label)
	}
	_, _ = fmt.Fprintf(
		out,
		"%s[%s] %s: %s",
		form.indent,
		label,
		printable.Line(hit.AnalyzerID),
		printable.Line(hit.Message),
	)
	if loc := location(hit); loc != "" {
		out.WriteString(" (" + loc + ")")
	}
	out.WriteByte('\n')

	if hit.Suggestion != "" {
		out.WriteString(form.arrow + "\u2192 " + printable.Line(hit.Suggestion) + "\n")
	}
	if form.withRule {
		// The full summary is a ~400 char paragraph that drowns the finding.
		if summary := strings.TrimSpace(hit.RuleSummary); summary != "" {
			rule := printable.Line(knowledge.FirstSentence(summary))
			out.WriteString(form.arrow + "rule: " + rule + "\n")
		}
	}
}

func location(hit finding.Finding) string {
	if hit.File == "" {
		return ""
	}
	if hit.Line > 0 {
		return fmt.Sprintf("%s:%d", printable.Path(hit.File), hit.Line)
	}
	return printable.Path(hit.File)
}
