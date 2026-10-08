package tsfacts

import (
	"slices"
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/scanner"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsdoc"
)

type docShape struct {
	start         int
	exempt, mixed bool
}

func (coll *collector) docComment(node *ast.Node) (docShape, bool) {
	ranges := slices.Collect(
		scanner.GetLeadingCommentRanges(coll.factory, coll.text, node.Pos()),
	)
	boundary := scanner.SkipTrivia(coll.text, node.Pos())
	first := len(ranges)
	for first > 0 && strings.Count(coll.text[ranges[first-1].End():boundary], "\n") <= 1 {
		first--
		boundary = ranges[first].Pos()
	}
	chain := ranges[first:]
	doc := slices.IndexFunc(chain, func(comment ast.CommentRange) bool {
		return tsdoc.IsJSDoc(coll.text[comment.Pos():comment.End()])
	})
	if doc < 0 {
		return docShape{}, false
	}
	for next := doc + 1; next < len(chain); next++ {
		if tsdoc.IsJSDoc(coll.text[chain[next].Pos():chain[next].End()]) {
			doc = next
		}
	}
	return coll.shapeOf(chain, doc), true
}

func (coll *collector) shapeOf(chain []ast.CommentRange, doc int) docShape {
	var others []string
	for idx, comment := range chain {
		if idx != doc {
			others = append(others, coll.text[comment.Pos():comment.End()])
		}
	}
	shape := tsdoc.Classify(coll.text[chain[doc].Pos():chain[doc].End()], others)
	return docShape{start: chain[doc].Pos(), exempt: shape.Exempt, mixed: shape.Mixed}
}
