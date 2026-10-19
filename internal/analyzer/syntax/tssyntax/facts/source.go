package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/scanner"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/tsparse"
)

type source struct {
	text  string
	lines tsparse.Lines
}

func newSource(text string) source {
	return source{text: text, lines: tsparse.NewLines(text)}
}

func (src source) lineAt(offset int) int {
	return src.lines.Of(scanner.SkipTrivia(src.text, offset))
}

func (src source) sourceOf(node *ast.Node) string {
	return src.text[scanner.SkipTrivia(src.text, node.Pos()):node.End()]
}

func (src source) nameOr(name *ast.Node, fallback string) string {
	if name == nil {
		return fallback
	}
	switch name.Kind {
	case ast.KindIdentifier, ast.KindPrivateIdentifier, ast.KindStringLiteral,
		ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		if text := name.Text(); text != "" {
			return text
		}
		return fallback
	}
	return src.sourceOf(name)
}
