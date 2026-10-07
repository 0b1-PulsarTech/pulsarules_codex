package gosyntax

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast/testfunc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
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
			found = append(found, syntax.DocSite{
				Symbol:         target.symbol,
				Kind:           target.kind,
				Line:           lineOf(fset, target.group.Pos()),
				Exempt:         target.exempt || isExemptDoc(target.group),
				MixedDirective: goast.HintFor(target.group, "") != "",
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

func isExemptDoc(group *ast.CommentGroup) bool {
	return goast.IsDirectiveGroup(group) ||
		goast.IsSimplificationGroup(group) ||
		isDeprecatedOnly(group)
}

func isDeprecatedOnly(group *ast.CommentGroup) bool {
	text := strings.TrimSpace(group.Text())
	const maxLines = 3
	return strings.HasPrefix(text, "Deprecated:") && !strings.Contains(text, "\n\n") &&
		strings.Count(text, "\n")+1 <= maxLines
}
