package simplification

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type Analyzer struct {
	langs    *pass.LanguageRegistry
	reporter pass.Reporter
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs: langs,
		reporter: pass.NewReporter(
			"simplification-path", pass.SeverityWarning, pass.CatSyntax,
		),
	}
}

func (analyzer *Analyzer) ID() string          { return "simplification-path" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if analyzer.langs == nil {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	extFilter := pass.ExtensionFilter(ctx, analyzer.ID(), pass.DefaultSourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && analyzer.langs.Lookup(fc.Extension) != nil
	}
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []pass.Finding {
			return scanFile(src, fc.Path, analyzer.langs.Lookup(fc.Extension), reporter)
		},
	)
}

func scanFile(src []byte, path string, lang pass.Language, reporter pass.Reporter) []pass.Finding {
	var findings []pass.Finding
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

func scanComments(lines []string, syntax lexer.CommentSyntax) []bool {
	scanner := lexer.NewCommentScanner(syntax)
	isComment := make([]bool, len(lines))
	for lineIdx, line := range lines {
		isComment[lineIdx] = scanner.IsCommentLine(line)
	}
	return isComment
}

func isMarkerLine(lang pass.Language, line string) bool {
	comment, ok := strings.CutPrefix(strings.TrimSpace(line), lang.CommentPrefix())
	if !ok {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(comment), lexer.SimplificationMarker)
}

func blockNamesUpgradePath(lang pass.Language, lines []string) bool {
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
