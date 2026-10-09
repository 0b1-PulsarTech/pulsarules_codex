package syntaxreport

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

type Analyzer struct {
	language syntax.Language
}

func NewAnalyzer(language syntax.Language) *Analyzer {
	return &Analyzer{language: language}
}

func (analyzer *Analyzer) ID() string { return string(analyzer.language) + "-syntax" }

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	findings := analyzer.unparsed(ctx)
	return append(findings, analyzer.unavailable(ctx)...)
}

func (analyzer *Analyzer) unparsed(ctx *pass.Context) []finding.Finding {
	reporter := finding.NewReporter(
		analyzer.ID()+"/unparsed",
		finding.SeverityWarning,
		finding.CatSyntax,
	)
	var findings []finding.Finding
	for _, fc := range ctx.ChangedFiles {
		file, ok := ctx.Syntax.Get(fc.Path)
		if !ok || fc.Generated || file.Language != analyzer.language || file.ParseError == "" {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path,
			reasonLine(file.ParseError),
			fmt.Sprintf("%s was not analysed: %s", fc.Path, tidy(file.ParseError)),
			"make the file parse within the limits; until it does, no style rule checks it",
		))
	}
	return findings
}

func (analyzer *Analyzer) unavailable(ctx *pass.Context) []finding.Finding {
	reporter := finding.NewReporter(
		analyzer.ID()+"/unavailable",
		finding.SeverityInfo,
		finding.CatSyntax,
	)
	var findings []finding.Finding
	for _, entry := range ctx.Syntax.Unavailable() {
		if entry.Language != analyzer.language {
			continue
		}
		message := fmt.Sprintf(
			"%d %s files not analysed: %s", entry.Files, entry.Language, tidy(entry.Reason))
		findings = append(findings, reporter.NewWithSuggestion(
			message,
			"fix the tool named in the reason and run again; these files were not checked",
		))
	}
	return findings
}

func tidy(reason string) string {
	const maxReasonBytes = 240

	line := printable.Line(strings.Join(strings.Fields(reason), " "))
	if len(line) <= maxReasonBytes {
		return line
	}
	cut := maxReasonBytes
	for cut > 0 && !utf8.RuneStart(line[cut]) {
		cut--
	}
	return line[:cut]
}

func reasonLine(reason string) int {
	// Go reasons carry "file:LINE:COL:" and the other providers carry "line LINE".
	if _, after, found := strings.Cut(reason, "line "); found {
		if line := leadingNumber(after); line > 0 {
			return line
		}
	}
	parts := strings.Split(reason, ":")
	for idx := 0; idx+1 < len(parts); idx++ {
		line, lineErr := strconv.Atoi(parts[idx])
		_, colErr := strconv.Atoi(parts[idx+1])
		if lineErr == nil && colErr == nil && line > 0 {
			return line
		}
	}
	return 0
}

func leadingNumber(text string) int {
	end := 0
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
	}
	line, err := strconv.Atoi(text[:end])
	if err != nil {
		return 0
	}
	return line
}
