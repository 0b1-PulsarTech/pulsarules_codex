package analysis

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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
		finding.AnalyzerID,
		finding.Message,
	)
	if loc := location(finding); loc != "" {
		out.WriteString(" (" + loc + ")")
	}
	out.WriteByte('\n')

	if finding.Suggestion != "" {
		out.WriteString(form.arrow + "\u2192 " + finding.Suggestion + "\n")
	}
	if form.withRule {
		if headline := firstFilledLine(finding.RuleBody); headline != "" {
			out.WriteString(form.arrow + "rule: " + headline + "\n")
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

func firstFilledLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func severityLabel(severity core.Severity) string {
	switch severity {
	case core.SeverityWarning:
		return "warn"
	case core.SeverityError:
		return "error"
	default:
		return "info"
	}
}
