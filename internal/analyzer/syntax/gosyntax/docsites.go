package gosyntax

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast/testfunc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/docshape"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type docTarget struct {
	symbol string
	kind   syntax.DocKind
	group  *ast.CommentGroup
	exempt bool
}

func docSites(fset *token.FileSet, source syntax.Source, file *ast.File) []syntax.DocSite {
	var found []syntax.DocSite
	for _, decl := range file.Decls {
		for _, target := range docTargets(decl, source, file) {
			if target.group == nil {
				continue
			}
			text := strings.Split(strings.TrimSpace(target.group.Text()), "\n")
			shape := docshape.Classify(
				lexer.DialectGo, commentLines(target.group), nil, text,
			)
			found = append(found, syntax.DocSite{
				Symbol:         target.symbol,
				Kind:           target.kind,
				Line:           lineOf(fset, target.group.Pos()),
				Exempt:         target.exempt || shape.Exempt,
				MixedDirective: shape.Mixed,
			})
		}
	}
	return found
}

func docTargets(decl ast.Decl, source syntax.Source, file *ast.File) []docTarget {
	switch node := decl.(type) {
	case *ast.FuncDecl:
		change := pass.FileChange{Path: source.Path, IsTest: source.IsTest}
		return []docTarget{{
			symbol: node.Name.Name,
			kind:   syntax.DocFunction,
			group:  node.Doc,
			exempt: testfunc.Covers(change, file, node),
		}}
	case *ast.GenDecl:
		if node.Tok == token.TYPE {
			return typeTargets(node)
		}
	}
	return nil
}

func commentLines(group *ast.CommentGroup) []string {
	lines := make([]string, 0, len(group.List))
	for _, comment := range group.List {
		lines = append(lines, strings.Split(comment.Text, "\n")...)
	}
	return lines
}
