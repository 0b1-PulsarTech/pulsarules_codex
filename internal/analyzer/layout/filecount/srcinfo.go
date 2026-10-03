package filecount

import (
	"bytes"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
)

type srcInfo struct {
	pkg  string
	skip bool
	ent  bool
}

func inspectSource(file string, src []byte) srcInfo {
	mode := parser.ParseComments | parser.PackageClauseOnly
	hasEnt := bytes.Contains(src, []byte(entImportPath))
	if hasEnt {
		mode = parser.ParseComments | parser.SkipObjectResolution
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), file, src, mode)
	if err != nil {
		return srcInfo{}
	}
	source := srcInfo{pkg: parsed.Name.Name, skip: isBuildIgnored(parsed)}
	if hasEnt {
		source.ent = inspectEnt(parsed)
	}
	return source
}

func isBuildIgnored(file *ast.File) bool {
	hasGoBuild, plusIgnored := false, false
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, comment := range group.List {
			switch {
			case constraint.IsGoBuild(comment.Text):
				hasGoBuild = true
				if isIgnoreTag(comment.Text) {
					return true
				}
			// A "// +build" line counts only outside the package doc.
			case group != file.Doc && constraint.IsPlusBuild(comment.Text):
				plusIgnored = plusIgnored || isIgnoreTag(comment.Text)
			}
		}
	}
	// The +build rule applies only without a //go:build line.
	return plusIgnored && !hasGoBuild
}

func isIgnoreTag(text string) bool {
	expr, err := constraint.Parse(text)
	return err == nil && isIgnoreExpr(expr)
}

func isIgnoreExpr(expr constraint.Expr) bool {
	switch typed := expr.(type) {
	case *constraint.TagExpr:
		return typed.Tag == "ignore"
	case *constraint.AndExpr:
		return isIgnoreExpr(typed.X) || isIgnoreExpr(typed.Y)
	}
	return false
}
