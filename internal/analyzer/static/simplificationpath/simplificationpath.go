package simplificationpath

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const markerPrefix = "simplification:"

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter("simplification-path", core.SeverityWarning, core.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "simplification-path" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		if fc.Extension != ".go" {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, scanFile(src, fc.Path, a.reporter)...)
	}
	return findings
}

func scanFile(src []byte, path string, reporter core.Reporter) []core.Finding {
	var findings []core.Finding
	lines := strings.Split(string(src), "\n")
	for lineIdx := 0; lineIdx < len(lines); lineIdx++ {
		if !isMarkerLine(lines[lineIdx]) {
			continue
		}
		blockEnd := lineIdx + 1
		for blockEnd < len(lines) && isCommentLine(lines[blockEnd]) {
			blockEnd++
		}
		if !blockNamesUpgradePath(lines[lineIdx:blockEnd]) {
			findings = append(findings, reporter.At(
				path,
				lineIdx+1,
				"simplification comment names no upgrade path",
				`add "Upgrade path: ..." or "revisit when ..." to the comment block`,
			))
		}
		lineIdx = blockEnd - 1
	}
	return findings
}

func isCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "//")
}

func isMarkerLine(line string) bool {
	comment, ok := strings.CutPrefix(strings.TrimSpace(line), "//")
	if !ok {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(comment), markerPrefix)
}

func blockNamesUpgradePath(lines []string) bool {
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		comment, _ := strings.CutPrefix(strings.TrimSpace(line), "//")
		words = append(words, strings.TrimSpace(comment))
	}
	// Joined with a space: golines wraps prose, and a label split across two lines is
	// still one label.
	block := strings.ToLower(strings.Join(words, " "))
	return strings.Contains(block, "upgrade path:") || strings.Contains(block, "revisit when")
}
