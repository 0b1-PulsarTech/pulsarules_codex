package simplification

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(
			"simplification-path", finding.SeverityWarning, finding.CatSyntax,
		),
	}
}

func (analyzer *Analyzer) ID() string             { return "simplification-path" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	extFilter := pass.ExtensionFilter(ctx, analyzer.ID(), language.SourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && language.ForExtension(fc.Extension).ID != ""
	}
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			return scanFile(src, fc.Path, language.ForExtension(fc.Extension), reporter)
		},
	)
}

func scanFile(
	src []byte,
	path string,
	lang language.Language,
	reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
	lines := strings.Split(string(src), "\n")
	// A block is the marker line plus every comment line immediately after it. One
	// scanner runs in file order first, so continuations match big-comment's reading.
	syntax := lang.Comments
	isComment := scanComments(lines, syntax)
	marker := lexer.CommentSyntax{Line: []string{lang.CommentPrefix()}}
	for lineIdx := 0; lineIdx < len(lines); lineIdx++ {
		if !lexer.IsSimplificationMarker(lines[lineIdx], marker) {
			continue
		}
		blockEnd := lineIdx + 1
		for blockEnd < len(lines) && isComment[blockEnd] {
			blockEnd++
		}
		if !blockNamesUpgradePath(syntax, lines[lineIdx:blockEnd]) {
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

func blockNamesUpgradePath(syntax lexer.CommentSyntax, lines []string) bool {
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		words = append(words, lexer.CommentBody(line, syntax))
	}
	// Joined with a space: golines wraps prose, and a label split across two lines is
	// still one label.
	block := strings.ToLower(strings.Join(words, " "))
	return strings.Contains(block, "upgrade path:") || strings.Contains(block, "revisit when")
}
