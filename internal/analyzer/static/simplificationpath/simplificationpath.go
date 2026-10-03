package simplificationpath

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type Analyzer struct {
	langs    *core.LanguageRegistry
	reporter core.Reporter
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs: langs,
		reporter: core.NewReporter(
			"simplification-path", core.SeverityWarning, core.CatSyntax,
		),
	}
}

func (a *Analyzer) ID() string          { return "simplification-path" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil {
		return nil
	}
	reporter := a.reporter.Resolved(ctx)
	extFilter := core.ExtensionFilter(ctx, a.ID(), core.DefaultSourceExtensions())
	eligible := func(fc core.FileChange) bool {
		return extFilter(fc) && a.langs.Lookup(fc.Extension) != nil
	}
	return core.EachChangedFile(
		ctx,
		eligible,
		func(fc core.FileChange, src []byte) []core.Finding {
			return scanFile(src, fc.Path, a.langs.Lookup(fc.Extension), reporter)
		},
	)
}

func scanFile(src []byte, path string, lang core.Language, reporter core.Reporter) []core.Finding {
	var findings []core.Finding
	lines := strings.Split(string(src), "\n")
	// A block is the marker line plus every comment line immediately after it. One
	// scanner runs in file order first, so continuations match big-comment's reading.
	isComment := scanComments(lines, lang.CommentSyntax())
	for lineIdx := 0; lineIdx < len(lines); lineIdx++ {
		if !isMarkerLine(lang, lines[lineIdx]) {
			continue
		}
		blockEnd := lineIdx + 1
		for blockEnd < len(lines) && isComment[blockEnd] {
			blockEnd++
		}
		if !blockNamesUpgradePath(lang, lines[lineIdx:blockEnd]) {
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

func scanComments(lines []string, syntax lexical.CommentSyntax) []bool {
	scanner := lexical.NewCommentScanner(syntax)
	isComment := make([]bool, len(lines))
	for lineIdx, line := range lines {
		isComment[lineIdx] = scanner.IsCommentLine(line)
	}
	return isComment
}

func isMarkerLine(lang core.Language, line string) bool {
	comment, ok := strings.CutPrefix(strings.TrimSpace(line), lang.CommentPrefix())
	if !ok {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(comment), lexical.SimplificationMarker)
}

func blockNamesUpgradePath(lang core.Language, lines []string) bool {
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		comment, _ := strings.CutPrefix(strings.TrimSpace(line), lang.CommentPrefix())
		words = append(words, strings.TrimSpace(comment))
	}
	// Joined with a space: golines wraps prose, and a label split across two lines is
	// still one label.
	block := strings.ToLower(strings.Join(words, " "))
	return strings.Contains(block, "upgrade path:") || strings.Contains(block, "revisit when")
}
