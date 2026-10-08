package tsparse

import (
	"fmt"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/parser"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/tspath"
)

func Parse(text string, kind core.ScriptKind) *ast.SourceFile {
	// The parser panics on a relative file name.
	name := "/source"
	options := ast.SourceFileParseOptions{
		FileName:         name,
		Path:             tspath.Path(name),
		JSDocParsingMode: ast.JSDocParsingModeParseNone,
	}
	return parser.ParseSourceFile(options, text, kind)
}

func FirstError(root *ast.SourceFile, text string) string {
	found := root.Diagnostics()
	if len(found) == 0 {
		return ""
	}
	line := NewLines(text).Of(found[0].Pos())
	return fmt.Sprintf("line %d: %s", line, found[0].Message())
}
