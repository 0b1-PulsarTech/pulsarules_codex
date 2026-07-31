package analysis

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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

// firstFilledLine is the rule body's headline. A body starts with a blank line
// after its frontmatter, so taking line one verbatim prints an empty "rule:".
func firstFilledLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// severityLabel returns the lowercase label for severity; the caller
// upper-cases it for the styles that want that (see writeFinding), so the
// case choice lives at the call site instead of a flag argument here.
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
